# UPP Publish Carousel

A microservice that continuously republishes content and annotations available in the native store.

## Code

publish-carousel

## Primary URL

<https://upp-prod-publish.ft.com/__publish-carousel/>

## Service Tier

Bronze

## Lifecycle Stage

Production

## Delivered By

content

## Supported By

content

## Known About By

- hristo.georgiev
- elina.kaneva
- robert.marinov
- tsvetan.dimitrov

## Host Platform

AWS

## Architecture

A microservice that continuously republishes content and annotations from the
native store. Checkout the project repository README for more details:
<https://github.com/Financial-Times/publish-carousel>

## Contains Personal Data

No

## Contains Sensitive Data

No

## Failover Architecture Type

ActiveActive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is deployed in both Publishing clusters. The failover guide for the cluster is located here: <https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

The release is triggered by making a Github release which is then picked up by a Jenkins multibranch pipeline. The Jenkins pipeline should be manually started in order for it to deploy the helm package to the Kubernetes clusters.

## Key Management Process Type

NotApplicable

## Key Management Details

There is no key rotation procedure for this system.

## Monitoring

Look for the pods in the cluster health endpoint and click to see pod health and checks:

- <https://upp-prod-publish-eu.upp.ft.com/__health/>
- <https://upp-prod-publish-us.upp.ft.com/__health/>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
