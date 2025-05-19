package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubelet/pkg/apis/credentialprovider/install"
	v1 "k8s.io/kubelet/pkg/apis/credentialprovider/v1"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	install.Install(scheme)
}

type ExecPlugin struct {
	plugin CredentialProvider
}

func NewCredentialProvider(plugin CredentialProvider) *ExecPlugin {
	return &ExecPlugin{plugin}
}

type CredentialProvider interface {
	GetCredentials(ctx context.Context, request *v1.CredentialProviderRequest, args []string) (response *v1.CredentialProviderResponse, err error)
}

func (e *ExecPlugin) Run(ctx context.Context) error {
	return e.runPlugin(ctx, os.Stdin, os.Stdout, os.Args[1:])
}

func (e *ExecPlugin) runPlugin(ctx context.Context, r io.Reader, w io.Writer, args []string) error {

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		gvk, err := json.DefaultMetaFactory.Interpret(data)

		if err != nil {
			return err
		}

		if gvk.GroupVersion() != v1.SchemeGroupVersion {
			return fmt.Errorf("group version %s is not supported", gvk.GroupVersion())
		}

		request, err := decodeRequest(data)
		if err != nil {
			return err
		}

		if request.Image == "" {
			return errors.New("image in plugin request was empty")
		}

		response, err := e.plugin.GetCredentials(ctx, request, args)

		if err != nil {
			return err
		}

		encodedResponse, err := encodeResponse(response)
		if err != nil {
			return err
		}

		writer := bufio.NewWriter(os.Stdout)
		defer writer.Flush()
		if _, err := writer.Write(encodedResponse); err != nil {
			return err
		}

	} else {
		return fmt.Errorf("Stdin data is missing. Please supply the program with the proper CredentialProviderRequest json")
	}

	return nil

}

func decodeRequest(data []byte) (*v1.CredentialProviderRequest, error) {
	obj, gvk, err := codecs.UniversalDecoder(v1.SchemeGroupVersion).Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "CredentialProviderRequest" {
		return nil, fmt.Errorf("kind was %q, expected CredentialProviderRequest", gvk.Kind)
	}

	if gvk.Group != v1.GroupName {
		return nil, fmt.Errorf("group was %q, expected %s", gvk.Group, v1.GroupName)
	}

	request, ok := obj.(*v1.CredentialProviderRequest)
	if !ok {
		return nil, fmt.Errorf("unable to convert %T to *CredentialProviderRequest", obj)
	}

	return request, nil
}

func encodeResponse(response *v1.CredentialProviderResponse) ([]byte, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q", mediaType)
	}

	encoder := codecs.EncoderForVersion(info.Serializer, v1.SchemeGroupVersion)
	data, err := runtime.Encode(encoder, response)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response: %v", err)
	}

	return data, nil
}
