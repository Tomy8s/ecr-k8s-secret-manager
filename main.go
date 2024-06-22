package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	ctx := context.TODO()
	k8sClient, err := newK8sClient()
	if err != nil {
		log.Fatal(err)
	}
	ecrClient, err := newECRClient(&ctx)
	if err != nil {
		log.Fatal(err)
	}
	secrets, err := buildSecrets(ecrClient, &ctx)
	if err != nil {
		log.Fatal(err)
	}
	createSecrets(secrets, k8sClient, &ctx)
}

func newK8sClient() (*kubernetes.Clientset, error) {
	var k8sConfig *rest.Config
	var err error
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		fmt.Println("Looks like we're in K8s!")
		k8sConfig, err = rest.InClusterConfig()
	} else {
		fmt.Println("Looks like we're not in K8s!")
		home, _ := os.UserHomeDir()
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", home+"/.kube/config")
	}
	if err != nil {
		fmt.Println("Failed to create k8s client")
		return nil, err
	}
	
	fmt.Println("Successfully created k8s client")
	return kubernetes.NewForConfig(k8sConfig)
}

func newECRClient(ctx *context.Context) (*ecr.Client, error) {
	cfg, err := config.LoadDefaultConfig(*ctx)
	if err != nil {
		return nil, err
	}

	return ecr.NewFromConfig(cfg), nil
}

func buildSecrets(ecrClient *ecr.Client, ctx *context.Context) ([]*v1.Secret, error) {
	output, err := ecrClient.GetAuthorizationToken(*ctx, &ecr.GetAuthorizationTokenInput{})

	if err != nil {
		return nil, err
	}
	secrets := make([]*v1.Secret, len(output.AuthorizationData))
	for i, authData := range output.AuthorizationData {
		// fmt.Println(*authData.ProxyEndpoint)
		// fmt.Println(authData.ExpiresAt, *authData.AuthorizationToken)

		dckCfg := DockerConfigJSON{
			Auths: map[string]DockerRegistryConfig{
				*authData.ProxyEndpoint: {
					Username: "AWS",
					Password: *authData.AuthorizationToken,
				},
			},
		}

		dckCfgJson, err := json.Marshal(dckCfg)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Println(string(dckCfgJson))
		
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr-registry-secret",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ecr-k8s-secret-manager",
				},
				Annotations: map[string]string{
					"ecr-k8s-secret-manager.io/expires-at": strconv.Itoa(int(authData.ExpiresAt.Unix())),
				},
			},
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				v1.DockerConfigJsonKey: dckCfgJson,
			},
		}
		
		fmt.Println(string(secret.String()))
		secrets[i] = secret
	}
	return secrets, nil
}

func createSecrets(secrets []*v1.Secret, k8sClient *kubernetes.Clientset, ctx *context.Context) []error {
	errors := make([]error, len(secrets))
	for i, secret := range secrets {
		_, errors[i] = k8sClient.CoreV1().Secrets("").Create(*ctx, secret, metav1.CreateOptions{})
		if errors[i] != nil {
			log.Fatal(errors[i])
		}
	}
	return errors
}
