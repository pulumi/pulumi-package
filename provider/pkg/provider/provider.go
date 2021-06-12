// Copyright 2016-2021, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-package/pkg/provider/component"
	"github.com/pulumi/pulumi-package/pkg/provider/invoke"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/logging"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	provSDK "github.com/pulumi/pulumi/sdk/v3/go/pulumi/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbempty "github.com/golang/protobuf/ptypes/empty"
	"github.com/pulumi/pulumi/pkg/v3/resource/provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
)

type PackageProvider struct {
	host       *provider.HostClient
	name       string
	version    string
	schema     []byte
	components map[string]component.Component
	invokes    map[string]invoke.InvokeFunc
}

// Serve launches the gRPC server for the resource provider.
func Serve(providerName, version string, schema []byte, components []component.Component, invokes []invoke.Invoke) {

	typeToComponent := map[string]component.Component{}
	for _, c := range components {
		typeToComponent[c.TypeToken] = c
	}
	typeToInvoke := map[string]invoke.InvokeFunc{}
	for _, i := range invokes {
		typeToInvoke[i.TypeToken] = i.InvokeF
	}

	err := provider.Main(providerName, func(host *provider.HostClient) (pulumirpc.ResourceProviderServer, error) {
		return &PackageProvider{
			host:       host,
			name:       providerName,
			version:    version,
			schema:     schema,
			components: typeToComponent,
			invokes:    typeToInvoke,
		}, nil
	})
	if err != nil {
		cmdutil.ExitError(err.Error())
	}
}

// Invoke dynamically executes a built-in function in the provider.
func (p *PackageProvider) Invoke(ctx context.Context,
	req *pulumirpc.InvokeRequest) (*pulumirpc.InvokeResponse, error) {
	label := fmt.Sprintf("%s.Invoke(%s)", p.name, req.Tok)
	logging.V(9).Infof("%s executing", label)

	invoke, ok := p.invokes[req.GetTok()]
	if !ok {
		return nil, errors.Errorf("unknown invoke request for token: %v", req.GetTok())
	}

	args, err := plugin.UnmarshalProperties(req.GetArgs(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.args", label), KeepUnknowns: true, SkipNulls: true, KeepSecrets: true,
	})
	if err != nil {
		return nil, err
	}

	outputs, err := invoke(args)
	if err != nil {
		return nil, errors.Wrapf(err, "invoke failed for token: %v", req.GetTok())
	}

	result, err := plugin.MarshalProperties(
		resource.NewPropertyMapFromMap(outputs),
		plugin.MarshalOptions{Label: fmt.Sprintf("%s.response", label), KeepUnknowns: true, SkipNulls: true},
	)
	if err != nil {
		return nil, err
	}
	return &pulumirpc.InvokeResponse{Return: result}, nil
}

// Construct creates a new instance of the provided component resource and returns its state.
func (p *PackageProvider) Construct(ctx context.Context,
	req *pulumirpc.ConstructRequest) (*pulumirpc.ConstructResponse, error) {

	comp, ok := p.components[req.GetType()]
	if !ok {
		return nil, errors.Errorf("unknown construct request for type: %v", req.GetType())
	}

	construct := func(ctx *pulumi.Context, typ, name string, inputs provSDK.ConstructInputs,
		options pulumi.ResourceOption) (*provSDK.ConstructResult, error) {
		args := comp.NewArgs()
		if err := inputs.CopyTo(args); err != nil {
			return nil, errors.Wrap(err, "setting args")
		}
		instance, err := comp.NewComponent(ctx, name, args, options)
		if err != nil {
			return nil, errors.Wrap(err, "creating component")
		}
		return provSDK.NewConstructResult(instance)
	}

	return provSDK.Construct(ctx, req, p.host.EngineConn(), construct)
}

func (p *PackageProvider) GetPluginInfo(context.Context, *pbempty.Empty) (*pulumirpc.PluginInfo, error) {
	return &pulumirpc.PluginInfo{
		Version: p.version,
	}, nil
}

// GetSchema returns the JSON-encoded schema for this provider's package.
func (p *PackageProvider) GetSchema(ctx context.Context,
	req *pulumirpc.GetSchemaRequest) (*pulumirpc.GetSchemaResponse, error) {
	if v := req.GetVersion(); v != 0 {
		return nil, errors.Errorf("unsupported schema version %d", v)
	}
	schema := string(p.schema)
	if schema == "" {
		schema = "{}"
	}
	return &pulumirpc.GetSchemaResponse{Schema: schema}, nil
}

// Configure configures the resource provider with "globals" that control its behavior.
func (p *PackageProvider) Configure(ctx context.Context,
	req *pulumirpc.ConfigureRequest) (*pulumirpc.ConfigureResponse, error) {
	return &pulumirpc.ConfigureResponse{
		AcceptSecrets:   true,
		SupportsPreview: true,
		AcceptResources: true,
	}, nil
}

// CheckConfig validates the configuration for this provider.
func (p *PackageProvider) CheckConfig(ctx context.Context,
	req *pulumirpc.CheckRequest) (*pulumirpc.CheckResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CheckConfig is not yet implemented")
}

// DiffConfig diffs the configuration for this provider.
func (p *PackageProvider) DiffConfig(ctx context.Context,
	req *pulumirpc.DiffRequest) (*pulumirpc.DiffResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DiffConfig is not yet implemented")
}

// StreamInvoke dynamically executes a built-in function in the provider. The result is streamed
// back as a series of messages.
func (p *PackageProvider) StreamInvoke(req *pulumirpc.InvokeRequest,
	server pulumirpc.ResourceProvider_StreamInvokeServer) error {
	return status.Error(codes.Unimplemented, "StreamInvoke is not yet implemented")
}

// Check validates that the given property bag is valid for a resource of the given type and returns
// the inputs that should be passed to successive calls to Diff, Create, or Update for this
// resource. As a rule, the provider inputs returned by a call to Check should preserve the original
// representation of the properties as present in the program inputs. Though this rule is not
// required for correctness, violations thereof can negatively impact the end-user experience, as
// the provider inputs are using for detecting and rendering diffs.
func (p *PackageProvider) Check(ctx context.Context, req *pulumirpc.CheckRequest) (*pulumirpc.CheckResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Check is not yet implemented")
}

// Diff checks what impacts a hypothetical update will have on the resource's properties.
func (p *PackageProvider) Diff(ctx context.Context, req *pulumirpc.DiffRequest) (*pulumirpc.DiffResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Diff is not yet implemented")
}

// Create allocates a new instance of the provided resource and returns its unique ID afterwards.
// (The input ID must be blank.)  If this call fails, the resource must not have been created (i.e.,
// it is "transactional").
func (p *PackageProvider) Create(ctx context.Context,
	req *pulumirpc.CreateRequest) (*pulumirpc.CreateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Create is not yet implemented")
}

// Read the current live state associated with a resource.  Enough state must be include in the
// inputs to uniquely identify the resource; this is typically just the resource ID, but may also
// include some properties.
func (p *PackageProvider) Read(ctx context.Context, req *pulumirpc.ReadRequest) (*pulumirpc.ReadResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Read is not yet implemented")
}

// Update updates an existing resource with new values.
func (p *PackageProvider) Update(ctx context.Context,
	req *pulumirpc.UpdateRequest) (*pulumirpc.UpdateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Update is not yet implemented")
}

// Delete tears down an existing resource with the given ID.  If it fails, the resource is assumed
// to still exist.
func (p *PackageProvider) Delete(ctx context.Context, req *pulumirpc.DeleteRequest) (*pbempty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Delete is not yet implemented")
}

// Cancel signals the provider to gracefully shut down and abort any ongoing resource operations.
// Operations aborted in this way will return an error (e.g., `Update` and `Create` will either a
// creation error or an initialization error). Since Cancel is advisory and non-blocking, it is up
// to the host to decide how long to wait after Cancel is called before (e.g.)
// hard-closing any gRPC connection.
func (p *PackageProvider) Cancel(context.Context, *pbempty.Empty) (*pbempty.Empty, error) {
	return &pbempty.Empty{}, nil
}
