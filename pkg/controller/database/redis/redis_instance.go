/*
Copyright 2019 The Crossplane Authors.

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

package redis

import (
	"context"
	"strconv"

	sdkerror "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/crossplane/provider-alibaba/apis/database/v1alpha1"
	"github.com/crossplane/provider-alibaba/pkg/controller/database"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/password"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	redisv1alpha1 "github.com/crossplane/provider-alibaba/apis/database/v1alpha1/redis"
	aliv1alpha1 "github.com/crossplane/provider-alibaba/apis/v1alpha1"
	"github.com/crossplane/provider-alibaba/pkg/clients/redis"
)

const (
	ErrCreateInstanceConnectionFailed = "cannot instance connection"

	// Default port of redis database
	defaultRedisPort = "6379"
)

// SetupRedisInstance adds a controller that reconciles RedisInstances.
func SetupRedisInstance(mgr ctrl.Manager, l logging.Logger) error {
	name := managed.ControllerName(v1alpha1.RedisInstanceGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&redisv1alpha1.RedisInstance{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.RedisInstanceGroupVersionKind),
			managed.WithExternalConnecter(&redisConnector{
				client:         mgr.GetClient(),
				usage:          resource.NewProviderConfigUsageTracker(mgr.GetClient(), &aliv1alpha1.ProviderConfigUsage{}),
				newRedisClient: redis.NewClient,
			}),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

type redisConnector struct {
	client         client.Client
	usage          resource.Tracker
	newRedisClient func(ctx context.Context, accessKeyID, accessKeySecret, region string) (redis.Client, error)
}

func (c *redisConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) { //nolint:gocyclo
	// account for the deprecated Provider type.
	cr, ok := mg.(*redisv1alpha1.RedisInstance)
	if !ok {
		return nil, errors.New(database.ErrNotInstance)
	}

	// provider has more than one kind of managed resource.
	var (
		sel    *xpv1.SecretKeySelector
		region string
	)
	switch {
	case cr.GetProviderConfigReference() != nil:
		if err := c.usage.Track(ctx, mg); err != nil {
			return nil, errors.Wrap(err, database.ErrTrackUsage)
		}

		pc := &aliv1alpha1.ProviderConfig{}
		if err := c.client.Get(ctx, types.NamespacedName{Name: cr.Spec.ProviderConfigReference.Name}, pc); err != nil {
			return nil, errors.Wrap(err, database.ErrGetProviderConfig)
		}
		if s := pc.Spec.Credentials.Source; s != xpv1.CredentialsSourceSecret {
			return nil, errors.Errorf(database.ErrFmtUnsupportedCredSource, s)
		}
		sel = pc.Spec.Credentials.SecretRef
		region = pc.Spec.Region
	case cr.GetProviderReference() != nil:
		p := &aliv1alpha1.Provider{}
		if err := c.client.Get(ctx, types.NamespacedName{Name: cr.Spec.ProviderReference.Name}, p); err != nil {
			return nil, errors.Wrap(err, database.ErrGetProvider)
		}
		sel = p.Spec.CredentialsSecretRef
		region = p.Spec.Region
	default:
		return nil, errors.New(database.ErrNoProvider)
	}

	if sel == nil {
		return nil, errors.New(database.ErrNoConnectionSecret)
	}

	s := &corev1.Secret{}
	nn := types.NamespacedName{Namespace: sel.Namespace, Name: sel.Name}
	if err := c.client.Get(ctx, nn, s); err != nil {
		return nil, errors.Wrap(err, database.ErrGetConnectionSecret)
	}

	redisClient, err := c.newRedisClient(ctx, string(s.Data["accessKeyId"]), string(s.Data["accessKeySecret"]), region)
	return &external{client: redisClient}, errors.Wrap(err, database.ErrCreateClient)
}

type external struct {
	client redis.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*redisv1alpha1.RedisInstance)
	if !ok {
		return managed.ExternalObservation{}, errors.New(database.ErrNotInstance)
	}

	if cr.Status.AtProvider.DBInstanceID == "" {
		return managed.ExternalObservation{}, nil
	}

	instance, err := e.client.DescribeDBInstance(cr.Status.AtProvider.DBInstanceID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(redis.IsErrorNotFound, err), database.ErrDescribeFailed)
	}

	cr.Status.AtProvider = redis.GenerateObservation(instance)
	var pw string
	switch cr.Status.AtProvider.DBInstanceStatus {
	case redisv1alpha1.RedisInstanceStateRunning:
		cr.Status.SetConditions(xpv1.Available())
		address, port, err := e.createConnectionIfNeeded(cr)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, ErrCreateInstanceConnectionFailed)
		}
		instance.Endpoint = &v1alpha1.Endpoint{
			Address: address,
			Port:    port,
		}

		pw, err = e.createAccountIfNeeded(cr)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, database.ErrCreateAccountFailed)
		}
	case redisv1alpha1.RedisInstanceStateCreating:
		cr.Status.SetConditions(xpv1.Creating())
	case redisv1alpha1.RedisInstanceStateDeleting:
		cr.Status.SetConditions(xpv1.Deleting())
	default:
		cr.Status.SetConditions(xpv1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: getConnectionDetails(pw, cr, instance),
	}, nil
}

func (e *external) createConnectionIfNeeded(cr *redisv1alpha1.RedisInstance) (string, string, error) {
	if cr.Spec.ForProvider.PubliclyAccessible {
		return e.createPublicConnectionIfNeeded(cr)
	}
	return e.createPrivateConnectionIfNeeded(cr)
}

func (e *external) createPrivateConnectionIfNeeded(cr *redisv1alpha1.RedisInstance) (string, string, error) {
	domain := cr.Status.AtProvider.DBInstanceID + ".redis.rds.aliyuncs.com"
	if cr.Spec.ForProvider.DBInstancePort == 0 {
		return domain, defaultRedisPort, nil
	}
	port := strconv.Itoa(cr.Spec.ForProvider.DBInstancePort)
	if cr.Status.AtProvider.ConnectionReady {
		return domain, port, nil
	}
	connectionDomain, err := e.client.ModifyDBInstanceConnectionString(cr.Status.AtProvider.DBInstanceID, cr.Spec.ForProvider.DBInstancePort)
	if err != nil {
		// The previous request might fail due to timeout. That's fine we will eventually reconcile it.
		if sdkErr, ok := err.(sdkerror.Error); ok {
			if sdkErr.ErrorCode() == "InvalidConnectionStringOrPort.Duplicate" {
				cr.Status.AtProvider.ConnectionReady = true
				return domain, port, nil
			}
		}
		return "", "", err
	}

	cr.Status.AtProvider.ConnectionReady = true
	return connectionDomain, port, nil
}

func (e *external) createPublicConnectionIfNeeded(cr *redisv1alpha1.RedisInstance) (string, string, error) {
	domain := cr.Status.AtProvider.DBInstanceID + "-pb.redis.rds.aliyuncs.com"
	if cr.Status.AtProvider.ConnectionReady {
		return domain, "", nil
	}
	port := defaultRedisPort
	if cr.Spec.ForProvider.DBInstancePort != 0 {
		port = strconv.Itoa(cr.Spec.ForProvider.DBInstancePort)
	}
	_, err := e.client.AllocateInstancePublicConnection(cr.Status.AtProvider.DBInstanceID, cr.Spec.ForProvider.DBInstancePort)
	if err != nil {
		// The previous request might fail due to timeout. That's fine we will eventually reconcile it.
		if sdkErr, ok := err.(sdkerror.Error); ok {
			if sdkErr.ErrorCode() == "InvalidConnectionStringOrPort.Duplicate" || sdkErr.ErrorCode() == "NetTypeExists" {
				cr.Status.AtProvider.ConnectionReady = true
				return domain, port, nil
			}
		}
		return "", "", err
	}

	cr.Status.AtProvider.ConnectionReady = true
	return domain, port, nil
}

func (e *external) createAccountIfNeeded(cr *redisv1alpha1.RedisInstance) (string, error) {
	if cr.Status.AtProvider.AccountReady {
		return "", nil
	}

	pw, err := password.Generate()
	if err != nil {
		return "", err
	}

	if cr.Spec.ForProvider.MasterUsername == "" {
		return pw, nil
	}

	err = e.client.CreateAccount(cr.Status.AtProvider.DBInstanceID, cr.Spec.ForProvider.MasterUsername, pw)
	if err != nil {
		// The previous request might fail due to timeout. That's fine we will eventually reconcile it.
		if sdkErr, ok := err.(sdkerror.Error); ok {
			if sdkErr.ErrorCode() == "InvalidAccountName.Duplicate" {
				cr.Status.AtProvider.AccountReady = true
				return "", nil
			}
		}
		return "", err
	}

	cr.Status.AtProvider.AccountReady = true
	return pw, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*redisv1alpha1.RedisInstance)
	if !ok {
		return managed.ExternalCreation{}, errors.New(database.ErrNotInstance)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.DBInstanceStatus == redisv1alpha1.RedisInstanceStateCreating {
		return managed.ExternalCreation{}, nil
	}

	req := redis.MakeCreateDBInstanceRequest(meta.GetExternalName(cr), &cr.Spec.ForProvider)
	instance, err := e.client.CreateDBInstance(req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, database.ErrCreateFailed)
	}

	// The crossplane runtime will send status update back to apiserver.
	cr.Status.AtProvider.DBInstanceID = instance.ID

	// Any connection details emitted in ExternalClient are cumulative.
	return managed.ExternalCreation{ConnectionDetails: getConnectionDetails("", cr, instance)}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*redisv1alpha1.RedisInstance)
	if !ok {
		return errors.New(database.ErrNotInstance)
	}
	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.DBInstanceStatus == redisv1alpha1.RedisInstanceStateDeleting {
		return nil
	}

	err := e.client.DeleteDBInstance(cr.Status.AtProvider.DBInstanceID)
	return errors.Wrap(resource.Ignore(redis.IsErrorNotFound, err), database.ErrDeleteFailed)
}

func getConnectionDetails(password string, cr *redisv1alpha1.RedisInstance, instance *redis.DBInstance) managed.ConnectionDetails {
	cd := managed.ConnectionDetails{
		xpv1.ResourceCredentialsSecretUserKey: []byte(instance.ID),
	}
	if cr.Spec.ForProvider.MasterUsername != "" {
		cd[xpv1.ResourceCredentialsSecretUserKey] = []byte(cr.Spec.ForProvider.MasterUsername)
	}
	if password != "" {
		cd[xpv1.ResourceCredentialsSecretPasswordKey] = []byte(password)
	}
	if instance.Endpoint != nil {
		cd[xpv1.ResourceCredentialsSecretEndpointKey] = []byte(instance.Endpoint.Address)
		cd[xpv1.ResourceCredentialsSecretPortKey] = []byte(instance.Endpoint.Port)
	}

	return cd
}
