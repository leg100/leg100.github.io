---
title: "Scaling Events in Go"
slug: scaling-events-in-go
date: 2024-10-15T12:35:02+01:00
tags:
  - go
  - postgres
draft: true
---

## 0. Synopsis

OTF is a framework for managing terraform. The backend is implemented as a go process, `otfd`. More than one `otfd` process can be run, forming an HA cluster. Postgres is the only dependency, used for not only persistence but for also for triggering and queuing events. The architecture is relatively simple, avoiding the overhead of relying on another service for events, e.g. redis. However, a large user of OTF has recently revealed that it is unresilient when under heavy loads, with the `otfd` process permanently ceasing to process further workloads until it is restarted. This article investigates the problem and enumerates over various solutions.

## 1. The Problem

An exposition of the problem relies upon first understanding how events are triggered and queued.

A Postgres `TRIGGER` triggers a function whenever an `INSERT`, `UPDATE`, or `DELETE` is carried out on a table. The function wraps information on the operation, including the table name, operation and primary key ID of the affected row, encoding it into JSON, before using `NOTIFY` to send the message to a global postgres queue named `events`.

A dedicated service in `otfd` called the "listener" uses `LISTEN` to subscribe to these event messages. Using the observer pattern, a number of brokers are registered with listener, with one broker for each table. The listener forwards the event onto the relevant broker and the broker uses the primary key ID in the event to retrieve the corresponding domain entity. For example, a broker for the "runs" table receives the event:

```json
{"table": "runs", "id": "run-123", "op": "INSERT"}
```

It uses that information to query the database and construct the go type `*Run` corresponding to ID `run-123`. The broker then wraps that domain entity into the corresponding domain event, e.g.:

```go
`*Event[*Run]{Type: Created, Payload: *Run}
```

The broker relays the domain event onto a number of subscribers. Each subscribers

`otfd` runs a number of "subsystems" responsible for performing cluster-wide duties:

* scheduler: manages workspace queues of pending runs
* listener: listens to a global postgres events queue, forwarding events to brokers
* logs proxy: manages a cache of "chunks" of terraform logs
* reporter: updates github, gitlab, etc with the status of terraform runs
* notifier: relays run events onto third parties, e.g. GCP pub/sub, slack, etc.
* job allocator: allocates terraform jobs to agents
* agent manager: manages agent registration and monitors health of agents
* agent daemon: carries out in-process terraform jobs

