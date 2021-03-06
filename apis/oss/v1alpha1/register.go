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

// Package v1alpha1 contains API Schema definitions for the oss v1alpha1 API group
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "oss.alibaba.crossplane.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// BucketKind is the kind of Bucket resource type
	BucketKind = reflect.TypeOf(Bucket{}).Name()
	// BucketGroupKind is the group and kind information of Bucket resource type
	BucketGroupKind = schema.GroupKind{Group: GroupVersion.Group, Kind: BucketKind}.String()
	// BucketKindAPIVersion is the kind and apiversion of Bucket resource type
	BucketKindAPIVersion = BucketKind + "." + GroupVersion.String()
	// BucketGroupVersionKind is the GVK of Bucket resource type
	BucketGroupVersionKind = GroupVersion.WithKind(BucketKind)
)

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}
