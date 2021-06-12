package component

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type Component struct {
	NewArgs      NewArgsFunc
	NewComponent NewComponentFunc
	TypeToken    string
}

type NewArgsFunc func() interface{}

type NewComponentFunc func(ctx *pulumi.Context,
	name string, args interface{}, opts ...pulumi.ResourceOption) (pulumi.ComponentResource, error)
