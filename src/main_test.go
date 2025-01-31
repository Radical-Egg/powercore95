package main

import (
	"os"
	"testing"

	"github.com/docker/docker/client"
)

func TestGetSandboxConfig(t *testing.T) {
	setEnvironmentVariables()
	os.Setenv("SANDBOX_INI_PATH", "./data/Sandbox.ini.default")
	ini, err := Ini2Struct()

	if err != nil {
		t.Error(err)
	}

	got, err := GetSandboxConfig("GlobalRecipeUnlocks", *ini)

	if err != nil {
		t.Error(err)
	}

	t.Log(got)
}

func TestWriteIni(t *testing.T) {
	setEnvironmentVariables()
	os.Setenv("SANDBOX_INI_PATH", "./data/Sandbox.ini.default")
	ini, err := Ini2Struct()

	if err != nil {
		t.Error(err)
	}

	writeIni(ini, "./test.ini")
}

func TestContainerAction(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	err = ContainerAction("restart", cli, "test-container")

	if err != nil {
		t.Error(err)
	}
}
