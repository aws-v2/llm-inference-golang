package worker

import (
	"context"
	"log"

	"github.com/moby/moby/client"
	"github.com/moby/moby/api/types/container"
)
type DockerClient struct {
	cli *client.Client
}

func NewDockerClient() *DockerClient {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	return &DockerClient{cli: cli}
}
func (d *DockerClient) RunWorker(modelID, modelPath string) {
	ctx := context.Background()

	resp, err := d.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: "python-layer:latest",
			Env: []string{
				"MODEL_ID=" + modelID,
				"MODEL_PATH=" + modelPath,
			},
		},
		HostConfig: nil,
		NetworkingConfig: nil,
		Platform: nil,
	})
	if err != nil {
		log.Println("Docker create error:", err)
		return
	}

	_, err = d.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})

	if err != nil {
		log.Println("Docker start error:", err)
		return
	}

	log.Println("Started worker container:", resp.ID)
}