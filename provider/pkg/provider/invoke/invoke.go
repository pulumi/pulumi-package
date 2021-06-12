package invoke

import "github.com/pulumi/pulumi/sdk/v3/go/common/resource"

type Invoke struct {
	TypeToken string
	InvokeF   InvokeFunc
}

type InvokeFunc func(args resource.PropertyMap) (map[string]interface{}, error)
