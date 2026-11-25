package main

import (
	"fmt"
	sidecar "github.com/Lmare/lightning-playground/sidecar"
)


const namespace = "lightning-playground"
const secretlnd = "lnd-credentials"
const mountedAPIStorage = "/app/nodes-storage"

func main() {
	fmt.Println("[Sidecar api] Starting")
	getLndSecretAndSaveThem()
	watchLndSecretModifiedAndSaveThem()
	fmt.Println("[Sidecar api] Complete -> go to sleep")
    select {}
}


// Get the lnd credentials and Store them
func getLndSecretAndSaveThem() {
	fmt.Println("[Sidecar api] getLndSecretAndSaveThem...")
	callback := &sidecar.Callback {
	    Context : map[string]interface{} {
			"namespace" : namespace,
			"secretName" : secretlnd,
			"mountedVolume" : mountedAPIStorage,
		},
		Chain : []func(*sidecar.Callback) error {
			sidecar.ReadSecret,
			sidecar.MountedVolumeToBasePath,
			sidecar.SecretToPath,
		},
	}
	if err := callback.CallNext(); err != nil {
		fmt.Println("[Sidecar api] getLndSecretAndSaveThem... Failed !")
		panic("error on getLndSecretAndSaveThem : %s" + err.Error())
	}
	fmt.Println("[Sidecar api] getLndSecretAndSaveThem... Done !")
}



// Get the lnd credentials and Store them
func watchLndSecretModifiedAndSaveThem() {
	fmt.Println("[Sidecar api] watchLndSecretModifiedAndSaveThem...")
	callback := &sidecar.Callback {
	    Context : map[string]interface{} {
			"namespace" : namespace,
			"secretName" : secretlnd,
			"mountedVolume" : mountedAPIStorage,
		},
		Chain : []func(*sidecar.Callback) error {
			sidecar.WatchSecretModified,
			sidecar.MountedVolumeToBasePath,
			sidecar.SecretToPath,
		},
	}
	if err := callback.CallNext(); err != nil {
		fmt.Println("[Sidecar api] watchLndSecretModifiedAndSaveThem... Failed !")
		panic("error on watchLndSecretModifiedAndSaveThem : %s" + err.Error())
	}
	fmt.Println("[Sidecar api] watchLndSecretModifiedAndSaveThem... Done !")
}
