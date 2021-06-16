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

package component

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const packageToken = "pulumi-package:index:Package"

var PackageComponent = Component{
	TypeToken:    packageToken,
	NewArgs:      NewPackageArgsFunc,
	NewComponent: NewStaticPageComponentFunc,
}

var NewPackageArgsFunc = func() interface{} {
	return &PackageArgs{}
}

var NewStaticPageComponentFunc = func(ctx *pulumi.Context,
	name string, args interface{}, opts ...pulumi.ResourceOption) (pulumi.ComponentResource, error) {
	return NewPackage(ctx, name, args.(*PackageArgs), opts...)
}

// The set of arguments for creating a StaticPage component resource.
type PackageArgs struct {
	// The HTML content for index.html.
	Language                    string             `pulumi:"language"`
	Name                        string             `pulumi:"name"`
	ServerBucketName            pulumi.StringInput `pulumi:"serverBucketName"`
	ServerBucketWebsiteEndpoint pulumi.StringInput `pulumi:"serverBucketWebsiteEndpoint"`
}

// The Package component resource.
type Package struct {
	pulumi.ResourceState

	Releases pulumi.StringArrayOutput `pulumi:"releases"`
}

// NewPackage creates a new Package component resource.
func NewPackage(ctx *pulumi.Context,
	name string, args *PackageArgs, opts ...pulumi.ResourceOption) (*Package, error) {
	if args == nil {
		args = &PackageArgs{}
	}

	component := &Package{}
	err := ctx.RegisterComponentResource(packageToken, name, component, opts...)
	if err != nil {
		return nil, err
	}

	iArgs := map[string]interface{}{
		"name":     args.Name,
		"language": args.Language,
	}
	iResult := map[string]interface{}{}

	conf := config.New(ctx, "pulumi-package")
	action := conf.Require("action")

	var newRelease pulumi.StringOutput = pulumi.String("").ToStringOutput()

	switch action {
	// TODO: clean, install_sdks
	case "new":
		err = ctx.Invoke("pulumi-package:index:New", iArgs, &iResult)
	case "generate":
		err = ctx.Invoke("pulumi-package:index:Generate", iArgs, &iResult)
	case "build":
		err = ctx.Invoke("pulumi-package:index:Build", iArgs, &iResult)
	case "install":
		err = ctx.Invoke("pulumi-package:index:Install", iArgs, &iResult)
	case "publish":

		newRelease = pulumi.All(args.ServerBucketName, args.ServerBucketWebsiteEndpoint).ApplyT(func(args []interface{}) (string, error) {
			version := conf.Require("version")
			iArgs["version"] = version
			iArgs["serverBucketName"] = args[0].(string)
			iArgs["serverBucketWebsiteEndpoint"] = args[1].(string)
			err = ctx.Invoke("pulumi-package:index:Publish", iArgs, &iResult)
			if err != nil {
				return "", err
			}
			return iResult["url"].(string), nil
		}).(pulumi.StringOutput)

	}

	if err != nil {
		return nil, err
	}

	component.Releases = pulumi.All(component.URN().ToStringOutput(), newRelease).ApplyT(func(a []interface{}) ([]string, error) {
		urn := a[0].(string)
		r := a[1].(string)
		args := struct {
			URN string `pulumi:"urn"`
		}{URN: urn}

		var result struct {
			State struct {
				Releases []string `pulumi:"releases"`
			} `pulumi:"state"`
		}

		tok := "pulumi:pulumi:getResource"
		if err := ctx.Invoke(tok, &args, &result); err != nil {
			return nil, fmt.Errorf("Invoke(%s): %w", tok, err)
		}
		releases := result.State.Releases
		if r != "" {
			releases = append(releases, r)
		}
		return releases, nil
	}).(pulumi.StringArrayOutput)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"releases": component.Releases,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
