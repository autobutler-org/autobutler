# ExoFlow

An abstract set of templates and automation to allow for packaging up a single codebase in a way it can run on multiple clouds.

The intent is to simplify and allow for multi-cloud multi-region failovers and deployments,
allowing for seamless reliable response.

## Outline

In a sense, I want to automate and generalize the deployment and configurations of micro-clouds.

### Microclouds

We have microservices and microfrontends in an attempt to segment and parallelize the effort of development and testing.
What about the need for something like the Ubuntu [microcloud](https://canonical.com/blog/canonical_releases_microcloud)?
We now spend much of our time, effort, and energy designing distributed infra systems that do one simple thing...run a container.

#### Same Code, Different Host

We need to boil down infra and deployments to their most basic components, and abstract out all the silly
and unnecessary decision-making that is being done. It should not matter if you run in AKS or EKS, the code should remain.

#### Reinventing the wheel

How do I reload secrets in my app? This is a solved problem. Why are we thinking about this still?

I need to scale my app on a custom metric. Again, this is a solved problem.

#### So, the interface

So the idea is like so, a single manifest, or bundle of manifests, that defines applications to deploy as networked microclouds.

If you want k8s, you can specify as such, and we can orchestrate across any of your selected/whitelisted providers.
If you want a minimalist build, we can deploy to an autoscaling group, or vm scaling set, or similar construct.

Simple example:

```yaml
name: demo-app
metadata:
  created: <FILL>
  provider:
    current: aws
    failover-chain:
    - azure
# Platform provider login specifics. When doing failovers, or initial deployments,
# it will utilize this config to deploy the compute to a platform
providers:
  aws:
    auth:
      AWS_PROFILE: env:AWS_PROFILE
  azure:
    auth:
      ARM_CLIENT_ID: env:ARM_CLIENT_ID
      ARM_CLIENT_SECRET: env:encrypted:ARM_CLIENT_SECRET
      ARM_TENANT_ID: env:ARM_TENANT_ID
# Defines attributes about the compute platform to deploy
compute:
  platform:
    # Affinity defines the preference of hosting order for the app,
    # meaning this most prefers available k8s, then vms,
    # then edge if neither is available.
    affinity:
    - k8s
    - vms
    - edge
app:
  container:
    image: docker.io/exokomodo/reformer
    tag: latest
    sync-tags: true
    edge-override:
      tag: edge-latest
  vm:
    image: s3://exokomodo-images/exokomodo/reformer.vmdk
    tag: latest
monitoring:
  # List of monitoring targets, so you can utilize and configure multiple monitoring platforms
  targets:
  - platform: grafana
    # NOTE: Missing is whatever config is needed to say, hey, put logs and metrics into grafana, and define some alerts
```

#### Reliability

If a microcloud fails, it should be automatic and trivial to redeploy or failover to another host. If data hosting fails
entirely, then a compatible snapshot should be spun up in a recovery host/region and the service proxies be made aware.

### Consumer Needs

#### Deployment of networked app

##### App - AWS

Cloudformation template to set up whatever IaC management you need to continue on.
Once that is up, deploy whatever components were requested (such as DB, object storage, open network ports, load balanced
routes, auto scaling group).

The idea would be to allow developers to write platform-agnostic code that would properly translate to its host platform.

##### App - Azure

Azure Resource Manager (ARM) template to set up whatever Infrastructure as Code (IaC) management you need to continue on.
Once that is up, deploy whatever components were requested (such as databases, object storage, open network ports, load
balanced routes, auto scaling group).

The idea would be to allow developers to write platform-agnostic code that would properly translate to its host platform.

#### Design Considerations

##### Proxy-Based API Translation

Rather than writing separate client library common frontends and figuring out dynamic dispatch on the run container, it
may be worth creating companion services that the cloud API requests target. It routes to the nearest one, which will
only know how to handle calls it is hosted on. For example, say I am using a message queue service for my app. In AWS
I want SQS and in Azure I want AQS, but in my code I want to write to the EQS (Exo Queue Service), which routes to an
instance of the `eqs-proxy`. Say my request originated on an AWS node and goes to the `eqs-proxy` on an AWS node, then
that call would assume and translate into the appropriate call. This would allow me to develop separate version of these
proxies, working on Azure and AWS support separate from one another, and allowing for separate testing. This would also
allow for an AWS node to target an `eqs-proxy` configured to target Azure, allowing for that node to not worry about
where the storage is located, but simply receive the results.
