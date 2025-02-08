# ExoFlow

An abstract set of templates and automation to allow for packaging up a single codebase in a way it can run on multiple clouds.

The intent is to simplify and allow for multi-cloud multi-region failovers and deployments,
allowing for seamless reliable response.

## Needs

### Deployment of networked app

#### App - AWS

Cloudformation template to set up whatever IaC management you need to continue on.
Once that is up, deploy whatever components were requested (such as DB, object storage, open network ports, load balanced
routes, auto scaling group).

The idea would be to allow developers to write platform-agnostic code that would properly translate to its host platform.

#### App - Azure

Azure Resource Manager (ARM) template to set up whatever Infrastructure as Code (IaC) management you need to continue on.
Once that is up, deploy whatever components were requested (such as databases, object storage, open network ports, load
balanced routes, auto scaling group).

The idea would be to allow developers to write platform-agnostic code that would properly translate to its host platform.

## Design Considerations

### Proxy-Based API Translation

Rather than writing separate client library common frontends and figuring out dynamic dispatch on the run container, it
may be worth creating companion services that the cloud API requests target. It routes to the nearest one, which will
only know how to handle calls it is hosted on. For example, say I am using a message queue service for my app. In AWS
I want SQS and in Azure I want AQS, but in my code I want to write to the EQS (Exo Queue Service), which routes to an
instance of the `eqs-proxy`. Say my request originated on an AWS node and goes to the `eqs-proxy` on an AWS node, then
that call would assume and translate into the appropriate call. This would allow me to develop separate version of these
proxies, working on Azure and AWS support separate from one another, and allowing for separate testing. This would also
allow for an AWS node to target an `eqs-proxy` configured to target Azure, allowing for that node to not worry about
where the storage is located, but simply receive the results.
