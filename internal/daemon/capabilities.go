package daemon

type Capabilities struct {
	ObserveSourceResults bool `json:"observeSourceResults"`
	SSHKeepAliveOptions  bool `json:"sshKeepAliveOptions"`
}

func currentCapabilities() Capabilities {
	return Capabilities{ObserveSourceResults: true, SSHKeepAliveOptions: true}
}
