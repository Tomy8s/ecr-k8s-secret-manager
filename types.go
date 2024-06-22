package main

type DockerRegistryConfig struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DockerConfigJSON struct {
	Auths map[string]DockerRegistryConfig `json:"auths"`
}
