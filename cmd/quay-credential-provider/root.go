/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	v1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"
)

var gitVersion string

type quayPlugin struct{}

const (
	CREDENTIAL_FILE_ENV_VAR = "CREDENTIAL_FILE"
)

type authConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

type configFile struct {
	AuthConfigs map[string]authConfig `json:"auths"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "quay-credential-provider",
	Short:   "Quay Credential Provider",
	Version: gitVersion,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		p := NewCredentialProvider(&quayPlugin{})

		if err := p.Run(context.TODO()); err != nil {
			klog.Errorf("Error within Quay Credential Plugin: %v", err)
			os.Exit(1)
		}

	},
}

// GetCredentials implements the logic to return credentials for the provided image
// Currently implemented for demonstration purposes and to be replaced with logic associated with Quay
func (q *quayPlugin) GetCredentials(ctx context.Context, image string, args []string) (response *v1.CredentialProviderResponse, err error) {

	authMap := map[string]v1.AuthConfig{}

	if credentialFile, ok := os.LookupEnv(CREDENTIAL_FILE_ENV_VAR); ok {

		if _, err := os.Stat(credentialFile); !os.IsNotExist(err) {
			var configFile configFile

			data, err := os.ReadFile(credentialFile)

			if err != nil {
				return nil, err
			}

			err = json.Unmarshal([]byte(data), &configFile)

			if err != nil {
				return nil, err
			}

			for registry, authConfig := range configFile.AuthConfigs {

				username := authConfig.Username
				password := authConfig.Password

				if username == "" {
					decoded, err := base64.StdEncoding.DecodeString(authConfig.Auth)
					if err != nil {
						return nil, fmt.Errorf("error decoding the auth for server: %s Error: %v", registry, err)
					}

					parts := strings.Split(string(decoded), ":")
					if len(parts) != 2 {
						return nil, fmt.Errorf("malformed auth for server: %s", registry)
					}

					username = parts[0]
					password = parts[1]

				}

				authMap[registry] = v1.AuthConfig{
					Username: username,
					Password: password,
				}
			}

		} else {
			klog.Warningf("Credential file at '%s' does not exist", credentialFile)
		}

	}

	return &v1.CredentialProviderResponse{
		CacheKeyType:  v1.RegistryPluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: 10 * time.Minute},
		Auth:          authMap,
	}, nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
