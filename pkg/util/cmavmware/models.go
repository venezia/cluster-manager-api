package cmavmware

const (
	VMWareProvider = "VMWare"
)

type CreateClusterInput struct {
	Name             string
	K8SVersion       string
	VMWare           VMWareSpec
	HighAvailability bool
	NetworkFabric    string
}

type CreateClusterOutput struct {
	Cluster ClusterItem
}

type GetClusterInput struct {
	Name string
}

type GetClusterOutput struct {
	Cluster ClusterDetailItem
}

type DeleteClusterInput struct {
	Name string
}

type DeleteClusterOutput struct {
	Status string
}

type ListClusterInput struct{}

type ListClusterOutput struct {
	Clusters []ClusterItem
}

type VMWareSpec struct {
	Namespace  string
	PrivateKey string
	Machines   []MachineSpec
}

type ClusterItem struct {
	ID     string
	Name   string
	Status string
}

type ClusterDetailItem struct {
	ID         string
	Name       string
	Status     string
	Kubeconfig string
}

type MachineSpec struct {
	Username            string
	Host                string
	Port                int
	ControlPlaneVersion string
}