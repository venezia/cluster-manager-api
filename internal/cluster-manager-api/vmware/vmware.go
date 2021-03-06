package vmware

import (
	pb "github.com/samsung-cnct/cluster-manager-api/pkg/generated/api"
	"github.com/samsung-cnct/cluster-manager-api/pkg/util/cmavmware"
	"github.com/samsung-cnct/cluster-manager-api/pkg/util/k8sutil/cma"
	"github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/sirupsen/logrus"
)

func (c *Client) CreateCluster(in *pb.CreateClusterMsg) (*pb.CreateClusterReply, error) {
	var controlPlaneNodes []cmavmware.MachineSpec
	var workerNodes []cmavmware.MachineSpec
	for _, j := range in.Provider.GetVmware().ControlPlaneNodes {
		var labels []cmavmware.KubernetesLabel
		for _, k := range j.Labels {
			labels = append(labels, cmavmware.KubernetesLabel{Name: k.Name, Value: k.Value})
		}
		controlPlaneNodes = append(controlPlaneNodes, cmavmware.MachineSpec{
			Host:     j.Host,
			Username: j.Username,
			Port:     int(j.Port),
			Password: j.Password,
			Labels:   labels,
		})
	}
	for _, j := range in.Provider.GetVmware().WorkerNodes {
		var labels []cmavmware.KubernetesLabel
		for _, k := range j.Labels {
			labels = append(labels, cmavmware.KubernetesLabel{Name: k.Name, Value: k.Value})
		}
		workerNodes = append(workerNodes, cmavmware.MachineSpec{
			Host:     j.Host,
			Username: j.Username,
			Port:     int(j.Port),
			Password: j.Password,
			Labels:   labels,
		})
	}
	result, err := c.cmaVMWareClient.CreateCluster(cmavmware.CreateClusterInput{
		Name:              in.Name,
		K8SVersion:        in.Provider.K8SVersion,
		ControlPlaneNodes: controlPlaneNodes,
		WorkerNodes:       workerNodes,
		APIEndpoint:       in.Provider.GetVmware().ApiEndpoint,
		HighAvailability:  in.Provider.HighAvailability,
		NetworkFabric:     in.Provider.NetworkFabric,
	})
	if err != nil {
		return &pb.CreateClusterReply{}, err
	}

	// Now going to create K8S CR
	err = c.cmaK8sClient.CreateCluster(in.Name, cmak8sutil.Cluster{
		CallbackURL: in.Callback.Url,
		Provider:    "vmware",
		RequestID:   in.Callback.RequestId,
	})

	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}

	return &pb.CreateClusterReply{
		Ok: true,
		Cluster: &pb.ClusterItem{
			Id:     result.Cluster.ID,
			Name:   result.Cluster.Name,
			Status: pb.ClusterStatus_PROVISIONING,
		},
	}, nil
}

func (c *Client) GetCluster(in *pb.GetClusterMsg) (*pb.GetClusterReply, error) {
	result, err := c.cmaVMWareClient.GetCluster(cmavmware.GetClusterInput{
		Name: in.Name,
	})
	if err != nil {
		return &pb.GetClusterReply{}, err
	}

	enumeratedStatus, found := pb.ClusterStatus_value[result.Cluster.Status]
	if !found {
		enumeratedStatus = 0
	}

	return &pb.GetClusterReply{
		Ok: true,
		Cluster: &pb.ClusterDetailItem{
			Id:         result.Cluster.ID,
			Name:       result.Cluster.Name,
			Status:     pb.ClusterStatus(enumeratedStatus),
			Kubeconfig: result.Cluster.Kubeconfig,
		},
	}, nil
}

func (c *Client) GetClusterList(in *pb.GetClusterListMsg) (*pb.GetClusterListReply, error) {
	var clusters []*pb.ClusterItem
	result, err := c.cmaVMWareClient.ListClusters(cmavmware.ListClusterInput{})
	if err != nil {
		return &pb.GetClusterListReply{}, err
	}
	for _, j := range result.Clusters {
		enumeratedStatus, found := pb.ClusterStatus_value[j.Status]
		if !found {
			enumeratedStatus = 0
		}
		clusters = append(clusters, &pb.ClusterItem{
			Id:     j.ID,
			Name:   j.Name,
			Status: pb.ClusterStatus(enumeratedStatus),
		})
	}
	return &pb.GetClusterListReply{
		Ok:       true,
		Clusters: clusters,
	}, nil
}

func (c *Client) DeleteCluster(in *pb.DeleteClusterMsg) (*pb.DeleteClusterReply, error) {
	result, err := c.cmaVMWareClient.DeleteCluster(cmavmware.DeleteClusterInput{
		Name: in.Name,
	})
	if err != nil {
		return &pb.DeleteClusterReply{}, err
	}

	// Now going to create K8S CR
	err = c.cmaK8sClient.UpdateOrCreateCluster(in.Name, cmak8sutil.Cluster{
		CallbackURL: in.Callback.Url,
		Provider:    in.Provider.String(),
		RequestID:   in.Callback.RequestId,
	})
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}
	err = c.cmaK8sClient.ChangeClusterStatus(in.Name, v1alpha1.ClusterPhaseDeleting)
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set AKS credentials into kubernetes, this is bad")
	}
	return &pb.DeleteClusterReply{
		Ok:     true,
		Status: result.Status,
	}, nil
}

func (c *Client) GetClusterUpgrades(in *pb.GetUpgradeClusterInformationMsg) (output *pb.GetUpgradeClusterInformationReply, err error) {
	result, err := c.cmaVMWareClient.GetClusterUpgrades(cmavmware.GetClusterUpgradesInput{
		Name: in.Name,
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetUpgradeClusterInformationReply{
		Ok:       true,
		Versions: result.Versions,
	}, nil

}

func (c *Client) ClusterUpgrade(in *pb.UpgradeClusterMsg) (output *pb.UpgradeClusterReply, err error) {
	_, err = c.cmaVMWareClient.ClusterUpgrade(cmavmware.ClusterUpgradeInput{
		Name:    in.Name,
		Version: in.Version,
	})
	if err != nil {
		return nil, err
	}

	// Now going to create K8S CR
	err = c.cmaK8sClient.UpdateOrCreateCluster(in.Name, cmak8sutil.Cluster{
		CallbackURL: in.Callback.Url,
		Provider:    in.Provider.String(),
		RequestID:   in.Callback.RequestId,
	})
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}
	err = c.cmaK8sClient.ChangeClusterStatus(in.Name, v1alpha1.ClusterPhaseUpgrading)
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}

	return &pb.UpgradeClusterReply{
		Ok: true,
	}, nil
}

func (c *Client) AdjustCluster(in *pb.AdjustClusterMsg) (*pb.AdjustClusterReply, error) {
	var addNodes []cmavmware.MachineSpec
	var removeNodes []cmavmware.RemoveMachineSpec
	for _, j := range in.GetVmware().AddNodes {
		var labels []cmavmware.KubernetesLabel
		for _, k := range j.Labels {
			labels = append(labels, cmavmware.KubernetesLabel{Name: k.Name, Value: k.Value})
		}
		addNodes = append(addNodes, cmavmware.MachineSpec{
			Host:     j.Host,
			Username: j.Username,
			Port:     int(j.Port),
			Password: j.Password,
			Labels:   labels,
		})
	}
	for _, j := range in.GetVmware().RemoveNodes {
		removeNodes = append(removeNodes, cmavmware.RemoveMachineSpec{
			Host: j.Host,
		})
	}
	_, err := c.cmaVMWareClient.AdjustCluster(cmavmware.AdjustClusterInput{
		Name:        in.Name,
		AddNodes:    addNodes,
		RemoveNodes: removeNodes,
	})

	// Now going to create K8S CR
	err = c.cmaK8sClient.UpdateOrCreateCluster(in.Name, cmak8sutil.Cluster{
		CallbackURL: in.Callback.Url,
		Provider:    in.Provider.String(),
		RequestID:   in.Callback.RequestId,
	})
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}
	err = c.cmaK8sClient.ChangeClusterStatus(in.Name, v1alpha1.ClusterPhaseUpgrading)
	if err != nil {
		// TODO Unsure what to do if we suddenly can't persist the credentials to kubernetes
		// TODO Going to log for now
		logrus.Errorf("Could not set Cluster CR into kubernetes, this is bad")
	}

	if err != nil {
		return &pb.AdjustClusterReply{}, err
	}
	return &pb.AdjustClusterReply{}, nil
}
