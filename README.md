# Publish Carousel
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/publish-carousel/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/publish-carousel?branch=master)

A microservice that continuously republishes content and annotations from the native store.

# API

See the Swagger YML [here](./api/api.yml) or the API Blueprint [here](./api/api.md).

# Developer Notes

The project is vendored using `govendor`. Please run:

```
govendor sync
```

before building and running the project locally.

To test the project, use:

```
govendor test -v -race +local
```

There are MongoDB and Etcd integration tests, which require local running instances of MongoDB and Etcd. These can be skipped (along with long running tests) by using the command:

```
govendor test -v -race -short +local
```

To connect to a MongoDB instance, please use the environment variable `MONGO_TEST_URL` i.e. `export MONGO_TEST_URL=localhost:27017`. To connect to an Etcd instance, please use the environment variable `ETCD_TEST_URL` i.e. `export ETCD_TEST_URL=http://localhost:2379`.

For running the Carousel locally, please see the command line arguments that need to be set using:

```
./publish-carousel --help
```

Please note that you must connect to the **Primary** Mongo instance if you are connecting to one of our UPP clusters.

# Code Structure

## Packages

The following packages have quite straightforward areas of responsibility:

* The `cms` package is responsible for making the POST calls to the `cms-notifier` in the required format.
* The `etcd` package is responsible for retrieving and watching keys in etcd.
* The `native` package is responsible for finding and reading documents from the `native-store` in Mongo.
* The `resources` package provides the services http endpoints.
* The `s3` package provides a high-level (reusable) package for reading and writing files to Amazon S3.

The `scheduler` and `tasks` packages are responsible for the general operation of the Carousel.

The `tasks` package provides an abstraction for the act of loading native content from the `native-store` (using the `native` package), and POST-ing it to `cms-notifier` using the `cms` package. In this way, the Carousel can be easily extended to support other tasks which require UUIDs from Mongo.

The `scheduler` package is responsible for the running of the Carousel, which is described in the overview to follow.

## Overview

At a high-level, the code has the notion of a **Scheduler**, a **Cycle**, and a **CycleMetadata**. Each **Cycle** can be in one of several **states** (which we will describe later).

The **Scheduler** is responsible for managing all the configured cycles. It exposes functions for adding, deleting, stopping, starting and resetting individual cycles, as well as general functions for shutting down all cycles.

A **Cycle** is responsible for continuously iterating over a subset of the native content in the `native-store`. One **Iteration** of the cycle is completed each time the cycle completes the republishing of that subset.

## Cycle Types

There are currently three different **Types** of cycle.

### ThrottledWholeCollection

The `ThrottledWholeCollection` type will iterate over an entire `native-store` collection (i.e. methode, wordpress), and republish the content at a configured **Throttle** (i.e. every 15 seconds).

### FixedWindow

The `FixedWindow` type will iterate over all results within a configured time window (i.e. within the last one hour). The `FixedWindow` cycle will dynamically adjust its throttle so that it will complete the republishing of all content from the last hour, before the _next_ hour begins.

> For example, if there are two items, the throttle will adjust itself to republish one item every 30 minutes.

The FixedWindow type, however, has a configured minimum throttle, which cannot be exceeded.

> For example, if the minimum throttle was configured to 1 minute, then if the cycle needs to republish 65 items, this will take one hour and 5 minutes.

In this case, the next time window will be adjusted to be 5 minutes longer, so that no items are missed for republishing.

### ScalingWindow

The `ScalingWindow` type will also iterate over all the results within a configured time window (like the `FixedWindow`). However, it has both a minimum throttle, and a maximum throttle configured, neither of which can be exceeded.

> For example, if the maximum throttle is configured to 1 minute, and there are 5 items to republish in the time window, then the cycle will take 5 minutes.

Once the cycle has completed, the next cycle iteration will adjust its time window to start where the previous cycle finished. This can mean that iterations of this cycle may have *very short* time windows.

If the time window is so short that there are no items to republish, then both the `ScalingWindow` and `FixedWindow` cycles have a configured **Cool Down** period (i.e. 5 minutes) which it will wait before starting the next iteration.

## Cycle Metadata

While a cycle iteration is in progress, the cycle collects and stores metadata about its progress within a **CycleMetadata** struct. The following data is tracked:

* The `total` number of items to republish in the iteration.
* The `completed` number of items republished so far.
* The derived `progress` through the iteration as a decimal percentage.
* The total number of republishes which have `errors`. An error can occur while parsing/loading the data from the `native-store`, or can occur while POST-ing to the `cms-notifier`.
* The current `iteration` of the cycle.
* The `currentUuid` that is being republished.
* The time window start (as `windowStart`). This is only for `ScalingWindow` and `FixedWindow` types.
* The time window end (as `windowEnd`). Also only for the time windowed types.
* The `states` of the cycle as an array. More about this later.

The Metadata which is tracked above is mostly used for informational purposes, and can be viewed in the Carousel UI, with the exception of the `completed` field.

When the Carousel is stopped, a shutdown hook will trigger, and the cycle's current CycleMetadata will be saved to S3 as a json file.

> This currently only happens for the ThrottledWholeCollection cycle type.

When the Carousel restarts, it will check S3 for the CycleMetadata file, and attempt to re-instate it.

> If the CycleMetadata is no longer compatible, then the Carousel will ignore it, and start the Cycle from the beginning again.

Once the Metadata is re-instated, the Cycle will take the number of completed items, and will [skip](https://docs.mongodb.com/manual/reference/method/cursor.skip/) that number within the mongo cursor.

> For example, if the Mongo Cursor contains 1000 items, and the CycleMetadata saved in S3 shows that 300 have been completed, then the cycle will skip the first 300 records, and start republishing from the 301st item in the cursor.

If the list of items to republish has grown between the time the iteration began, and the time the process is restarted, we may not pick up exactly where we left off, but we should be *close enough* to where we were before.

This works because when an item is persisted in Mongo, it auto-generates an `_id`, and all queries which are *not* sorted are naturally ordered by this `_id`.

The items which have been added to the list will be republished on the next iteration, but they should also be republished during the shorter time windowed cycles, unless the Carousel is stopped for an extended period of time.

## Cycle States

The cycle can be in several **States**:

* **Starting**: the cycle is preparing to begin its initial iteration, after being Stopped.
* **Running**: the cycle is processing an iteration.
* **Stopped**: the cycle is no longer processing, and needs to be started.
* **Cooldown**: the cycle is waiting between iterations, due to a lack of items to republish.
* **Unhealthy**: the cycle has experienced an issue during normal processing.

A cycle can be in several states, but most of them are mutually exclusive, with the exception of the Unhealthy state, which can accompany any of them. Cycles, however, can currently only become unhealthy due to connectivity issues with Mongo, which interrupt the processing of the iteration.

> For the initial version of the Carousel, in all cases of a cycle becoming unhealthy, the cycle will **stop**. This is subject to change.

## Active / Passive

The Carosuel will run in the Publishing Cluster, which is an Active/Passive environment. As a result, the Carousel will also run in an Active/Passive manner, and will be disabled by default in the Passive region.

The Carousel, however, will **not** be automatically started during a failover scenario, and will remain passive. This is to ensure we do not overload the Cluster, which could potentially exacerbate any problems within the Publishing environment.

The Carousel uses the etcd key `/ft/config/publish-carousel/enable` to determine whether or not it needs to be in the Active or Passive modes on startup. If this toggle changes at any time, the Carousel will shutdown or startup as required.

## Configuration

On startup, the Carousel will read cycle configuration from a provided YAML file, add them to the Scheduler, attempt to restore the previous state from S3, and start them up. To configure cycles, the following fields are **required** for all cycle types:

* `name`: The name of the cycle.
* `type`: The type - can be one of `ThrottledWholeCollection`, `ScalingWindow`, `FixedWindow`.
* `origin`: The Origin System ID to use when POST-ing to the `cms-notifier`.
* `collection`: The `native-store` collection to retrieve content from.
* `coolDown`: The time between iterations. N.B. this is currently required for all cycle types.

The ThrottledWholeCollection type requires one additional field:

* `throttle`: The interval between each republish.

The ScalingWindow and FixedWindow types require the following additional fields:

* `timeWindow`: The time period to republish for (i.e. one hour).
* `minimumThrottle`: The lower bound for the computed throttle.

And finally, the ScalingWindow requires one extra field:

* `maximumThrottle`: The upper bound for the computed throttle.

# Developers on Windows

The Publish Carousel writes a metadata file to S3 on a graceful shutdown. Unfortunately, this functionality does not work on Windows using Git Bash, but does work when using the Command Prompt.

It works as expected on a Mac.
