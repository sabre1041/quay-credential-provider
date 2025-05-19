package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	v1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"
)

var gitVersion string

type quayPlugin struct{}

const (
	FederatedRobotAccountAnnotation    = "quay.io/federated-robot-account"
	QuayHostOverrideAnnotation         = "quay.io/host-override"
	QuayIgnoreTLSVerifyAnnotation      = "quay.io/ignore-tls-verify"
	QuayCertificateFileAnnotation      = "quay.io/certificate-file"
	QuayPlainHttpAnnotation            = "quay.io/plain-http"
	DefaultQuayHost                    = "quay.io"
	DefaultQuayRobotFederationEndpoint = "/oauth2/federation/robot/token"
	CREDENTIAL_FILE_ENV_VAR            = "CREDENTIAL_FILE"
)

type authConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

type configFile struct {
	AuthConfigs map[string]authConfig `json:"auths"`
}

type quayTokenResponse struct {
	Token string `json:"token"`
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
func (q *quayPlugin) GetCredentials(ctx context.Context, request *v1.CredentialProviderRequest, args []string) (response *v1.CredentialProviderResponse, err error) {

	if len(request.ServiceAccountToken) == 0 {
		return nil, nil
	}

	image := request.Image

	// Verify and process ServiceAccount Annotations
	federatedRobotAccount, ok := request.ServiceAccountAnnotations[FederatedRobotAccountAnnotation]
	if !ok || len(federatedRobotAccount) == 0 {
		return nil, fmt.Errorf("federated robot account annotation %s is not found or the value is empty", FederatedRobotAccountAnnotation)
	}

	// Check if image has scheme
	if !strings.Contains(image, "://") {
		image = "https://" + image
	}

	imageUrl, err := url.Parse(image)
	if err != nil {
		return nil, fmt.Errorf("error parsing image reference %s: %v", request.Image, err)
	}

	quayFederationCredentialURL := &url.URL{
		Scheme: "https",
		Host:   imageUrl.Hostname(),
		Path:   DefaultQuayRobotFederationEndpoint,
	}

	quayHostOverride, ok := request.ServiceAccountAnnotations[QuayHostOverrideAnnotation]
	if ok && len(quayHostOverride) > 0 {
		quayFederationCredentialURL.Host = quayHostOverride
	}

	plainHttpAnnotation := extractBoolAnnotation(request.ServiceAccountAnnotations, QuayPlainHttpAnnotation)
	if plainHttpAnnotation {
		quayFederationCredentialURL.Scheme = "http"
	}

	var tlsClientConfig = tls.Config{}

	ignoreTlsVerifyAnnotation := extractBoolAnnotation(request.ServiceAccountAnnotations, QuayPlainHttpAnnotation)

	if ignoreTlsVerifyAnnotation {
		tlsClientConfig.InsecureSkipVerify = true
	}

	// Build HTTP Client
	tr := &http.Transport{
		TLSClientConfig: &tlsClientConfig,
	}

	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(http.MethodGet, quayFederationCredentialURL.String(), http.NoBody)
	req.SetBasicAuth(federatedRobotAccount, request.ServiceAccountToken)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {

		if resBody != nil {
			return nil, fmt.Errorf("error getting credentials from Quay: Status Code: %d - Error Message: %s", resp.StatusCode, string(resBody))
		}

		return nil, fmt.Errorf("error getting credentials from Quay: %d", resp.StatusCode)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading response from Quay: %d - Error: %v", resp.StatusCode, err)
	}

	quayTokenResponse := quayTokenResponse{}

	if err := json.Unmarshal(resBody, &quayTokenResponse); err != nil {
		return nil, err
	}

	authMap := map[string]v1.AuthConfig{}

	authMap[imageUrl.Hostname()] = v1.AuthConfig{
		Username: federatedRobotAccount,
		Password: quayTokenResponse.Token,
	}

	jp := jwt.NewParser()
	claims := jwt.RegisteredClaims{}
	jwt, _, err := jp.ParseUnverified(quayTokenResponse.Token, &claims)
	if err != nil {
		log.Fatalf("Error parsing Quay JWT: %v\n", err)
	}

	expirationTime, _ := jwt.Claims.GetExpirationTime()
	duration := expirationTime.Time.Sub(time.Now())

	return &v1.CredentialProviderResponse{
		CacheKeyType:  v1.RegistryPluginCacheKeyType,
		CacheDuration: &metav1.Duration{Duration: duration},
		Auth:          authMap,
	}, nil
}

func extractBoolAnnotation(annotations map[string]string, key string) bool {

	if value, ok := annotations[key]; ok && len(value) > 0 {
		boolVal, err := strconv.ParseBool(value)

		if err != nil {
			return boolVal
		}

	}
	return false
}

func getCacheDuration(jwtToken string) (*metav1.Duration, error) {

	cacheDuration := &metav1.Duration{Duration: 10 * time.Minute}

	jp := jwt.NewParser()
	claims := jwt.RegisteredClaims{}
	jwt, _, err := jp.ParseUnverified(jwtToken, &claims)
	if err != nil {
		return cacheDuration, nil
	}

	expirationTime, _ := jwt.Claims.GetExpirationTime()
	cacheDuration = &metav1.Duration{Duration: expirationTime.Time.Sub(time.Now())}

	return cacheDuration, nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
