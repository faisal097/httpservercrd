package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func parseK8sYaml(ctx context.Context, fileR []byte) []runtime.Object {
	log := log.FromContext(ctx)
	acceptedK8sTypes := regexp.MustCompile(`(Namespace|Deployment|Service)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(f), nil, nil)

		if err != nil {
			log.Error(err, "Error while decoding YAML object")
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.Info(fmt.Sprintf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind))
		} else {
			retVal = append(retVal, obj)
		}

	}
	return retVal
}

func stringify(ctx context.Context, object interface{}) string {
	log := log.FromContext(ctx)
	bytes, err := json.Marshal(object)
	if err != nil {
		log.Error(err, "Error while json.Marshal object")
		return ""
	}

	return string(bytes)
}
