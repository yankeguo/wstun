[
  {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: 'k8s-api-relay-user-admin',
      namespace: 'default',
    },
  },
  {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: 'k8s-api-relay-user-admin-token',
      namespace: 'default',
      annotations: {
        'kubernetes.io/service-account.name': 'k8s-api-relay-user-admin',
      },
    },
    type: 'kubernetes.io/service-account-token',
  },
  {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'ClusterRoleBinding',
    metadata: {
      name: 'k8s-api-relay-user-admin',
    },
    roleRef: {
      apiGroup: 'rbac.authorization.k8s.io',
      kind: 'ClusterRole',
      name: 'cluster-admin',
    },
    subjects: [
      {
        kind: 'ServiceAccount',
        name: 'k8s-api-relay-user-admin',
        namespace: 'default',
      },
    ],
  },
  {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: 'k8s-api-relay',
      namespace: 'default',
    },
    type: 'Opaque',
    stringData: {
      'config.yaml': std.manifestYamlDoc({
        server_url: std.extVar('SERVER_URL'),
        service_name: std.extVar('SERVICE_NAME'),
        service_token: std.extVar('SERVICE_TOKEN'),
        target_addr: std.extVar('TARGET_ADDR'),
        tls_insecure: false,
        reconnect_interval: '5s',
        max_reconnect_attempts: 0,
        log_level: 'info',
      }, indent_array_in_object=true, quote_keys=false),
    },
  },
  {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: 'k8s-api-relay',
      namespace: 'default',
    },
    spec: {
      clusterIP: 'None',
      selector: {
        app: 'k8s-api-relay',
      },
      ports: [
        {
          name: 'placeholder',
          protocol: 'TCP',
          port: 42,
          targetPort: 42,
        },
      ],
    },
  },
  {
    apiVersion: 'apps/v1',
    kind: 'StatefulSet',
    metadata: {
      name: 'k8s-api-relay',
      namespace: 'default',
    },
    spec: {
      serviceName: 'k8s-api-relay',
      replicas: 1,
      selector: {
        matchLabels: {
          app: 'k8s-api-relay',
        },
      },
      template: {
        metadata: {
          labels: {
            app: 'k8s-api-relay',
          },
        },
        spec: {
          containers: [
            {
              name: 'wstun-client',
              image: 'docker.cnb.cool/yankeguo/wstun:client-20251104095534-f2d2954',
              imagePullPolicy: 'IfNotPresent',
              volumeMounts: [
                {
                  name: 'config',
                  mountPath: '/wstun-client',
                  readOnly: true,
                },
              ],
              resources: {
                requests: {
                  cpu: '100m',
                  memory: '64Mi',
                },
                limits: {
                  cpu: '500m',
                  memory: '128Mi',
                },
              },
            },
          ],
          volumes: [
            {
              name: 'config',
              secret: {
                secretName: 'k8s-api-relay',
                defaultMode: 400,
              },
            },
          ],
          restartPolicy: 'Always',
        },
      },
    },
  },
]
