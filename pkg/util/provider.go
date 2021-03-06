/*
Copyright 2021 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crossplanev1alpha1 "github.com/crossplane/provider-alibaba/apis/v1alpha1"
)

const (
	// AccessKeyID is Alibaba Cloud Access key ID
	AccessKeyID = "accessKeyId"
	// AccessKeySecret is Alibaba Cloud Access Secret Key
	AccessKeySecret = "accessKeySecret"
	// SecurityToken is Alibaba Cloud STS token
	SecurityToken = "securityToken"
)

// GetProviderConfig gets ProviderConfig
func GetProviderConfig(ctx context.Context, k8sClient client.Client, providerConfigName string) (*crossplanev1alpha1.ProviderConfig, error) {
	providerConfig := &crossplanev1alpha1.ProviderConfig{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: providerConfigName}, providerConfig); err != nil {
		return nil, fmt.Errorf("failed to get ProviderConfig: %v", err)
	}
	return providerConfig, nil
}
