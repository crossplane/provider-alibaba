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

package v1alpha1

import (
	runtimev1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// CLBList contains a list of CLB
type CLBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLB `json:"items"`
}

// +kubebuilder:object:root=true

// CLB is a managed resource that represents an CLB instance
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,alibaba},shortName=redis
type CLB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBSpec   `json:"spec,omitempty"`
	Status CLBStatus `json:"status,omitempty"`
}

// CLBSpec defines the desired state of CLB
type CLBSpec struct {
	runtimev1.ResourceSpec `json:",inline"`
	ForProvider            CLBParameter `json:"forProvider"`
}

// CLBStatus defines the observed state of CLB
type CLBStatus struct {
	runtimev1.ResourceStatus `json:",inline"`
	AtProvider               CLBObservation `json:"atProvider,omitempty"`
}

// CLBParameter is the isolated place to store files
type CLBParameter struct {
	// Region is the ID of the region where you want to create the SLB instance.
	Region *string `json:"region"`
	// AddressType is the type of IP address that the SLB instance uses to provide services. Valid values:
	// internet: After an Internet-facing SLB instance is created, the system assigns a public IP address to the SLB instance.
	// Then, the SLB instance can forward requests from the Internet.
	// intranet: After an internal-facing SLB instance is created, the system assigns a private IP address to the SLB instance.
	// Then, the SLB instance can forward only internal requests.
	AddressType *string `json:"addressType,omitempty"`

	// Address is the IP address
	Address *string `json:"address,omitempty"`

	// Bandwidth is the maximum bandwidth value of the listener. Unit: Mbit/s.
	// Valid values: -1 and 1 to 5120.
	// -1: For a pay-by-data-transfer Internet-facing SLB instance, you can set the value to -1. This indicates that
	// the bandwidth is unlimited.
	// 1 to 5120: For a pay-by-bandwidth Internet-facing SLB instance, you can specify a bandwidth cap for each listener.
	// The sum of bandwidth limit values of all listeners cannot exceed the maximum bandwidth value of the SLB instance.
	Bandwidth *int32 `json:"bandwidth,omitempty"`

	// InternetChargeType is the metering method of the Internet-facing SLB instance. Valid values:
	// paybytraffic (default): pay-by-data-transfer
	// +kubebuilder:default:=paybytraffic
	InternetChargeType *string `json:"internetChargeType,omitempty"`

	// VpcID is the ID of the virtual private cloud (VPC) to which the SLB instance belongs.
	VpcID *string `json:"vpcId,omitempty"`

	// VSwitchID is the ID of the vSwitch to which the SLB instance is attached.
	// To create an SLB instance that is deployed in a VPC, you must set this parameter. If you specify this parameter,
	// the value of the AddressType parameter is set to intranet by default.
	VSwitchID *string `json:"vSwitchId,omitempty"`

	// LoadBalancerSpec is the specification of the SLB instance.
	// The types of SLB instance that you can create vary by region.
	// +kubebuilder:validation:Enum:=slb.s1.small;slb.s2.small;slb.s2.medium;slb.s3.small;slb.s3.medium;slb.s3.large
	LoadBalancerSpec *string `json:"loadBalancerSpec,omitempty"`

	// ClientToken that is used to ensure the idempotence of the request. You can use the client to generate the value,
	// but you must ensure that it is unique among different requests. The token can contain only ASCII characters and
	// cannot exceed 64 characters in length.
	ClientToken *string `json:"clientToken,omitempty"`

	OwnerID                      *int64  `json:"ownerId,omitempty"`
	ResourceOwnerAccount         *string `json:"resourceOwnerAccount,omitempty"`
	ResourceOwnerID              *int64  `json:"resourceOwnerId,omitempty"`
	OwnerAccount                 *string `json:"ownerAccount,omitempty"`
	MasterZoneID                 *string `json:"masterZoneId,omitempty"`
	SlaveZoneID                  *string `json:"slaveZoneId,omitempty"`
	ResourceGroupID              *string `json:"resourceGroupId,omitempty"`
	PayType                      *string `json:"payType,omitempty"`
	PricingCycle                 *string `json:"pricingCycle,omitempty"`
	Duration                     *int32  `json:"duration,omitempty"`
	AutoPay                      *bool   `json:"autoPay,omitempty"`
	AddressIPVersion             *string `json:"addressIPVersion,omitempty"`
	DeleteProtection             *string `json:"deleteProtection,omitempty"`
	ModificationProtectionStatus *string `json:"modificationProtectionStatus,omitempty"`
	ModificationProtectionReason *string `json:"modificationProtectionReason,omitempty"`
}

// CLBObservation is the representation of the current state that is observed.
type CLBObservation struct {
	LoadBalancerID               *string `json:"loadBalancerID,omitempty"`
	CreateTime                   *string `json:"CreateTime,omitempty"`
	NetworkType                  *string `json:"NetworkType,omitempty"`
	MasterZoneID                 *string `json:"MasterZoneId,omitempty"`
	ModificationProtectionReason *string `json:"ModificationProtectionReason,omitempty"`
	ModificationProtectionStatus *string `json:"ModificationProtectionStatus,omitempty"`
	LoadBalancerStatus           *string `json:"LoadBalancerStatus,omitempty"`
	ResourceGroupID              *string `json:"ResourceGroupId,omitempty"`
	DeleteProtection             *string `json:"DeleteProtection,omitempty"`
	// Though `Address` is one of the Parameter, but if the parameter it's not set, it still can be generated.
	Address *string `json:"address,omitempty"`
}
