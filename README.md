# Service utilities

Various utils for building services with etcd

## Overview

The `svcutil` package provides utilities for building distributed services using etcd, including:

- Service registration and discovery
- Distributed locking and coordination
- Lease management for resource allocation
- IP and ID range handling

## Core Components

### Service

The `Service` class provides the main interface to etcd and handles configuration management, locks, and service registration.

```go
// Create a new Service instance
svc, err := svcutil.NewService(
    svcutil.Name("user-endpoint"),
    svcutil.Scope("history"),
    svcutil.EtcdEndpoints("localhost:2379"),
    svcutil.LeaseTTL(30)
)
```

#### Key Features

- **Configuration Management**: Load service and host-specific configurations from etcd
- **Distributed Locking**: Acquire and release locks for coordination between services
- **Service Identification**: Generate unique service IDs across your cluster

#### Methods

- `NewService(options...)`: Creates a new Service instance with the provided options
- `Close()`: Gracefully shuts down the Service
- `AcquireLock(ctx, name)`: Acquires a named distributed lock. Locks are not guaranteed to survive if connection to etcd has been lost, use leases instead.
- `ReleaseLock(ctx, name)`: Releases a previously acquired lock
- `LoadConfig(ctx, configurationType, cfg)`: Loads configuration from etcd
- `ID(id)`: Creates an ID structure that identifies this service instance

### Lease

The `Lease` class provides resource leasing functionality, enabling exclusive access to IDs or IPs from a predefined range.

```go
// Create a lease for an ID range
idRange, _ := svcutil.NewIDRange("1-5")
lease := svcutil.NewLease(idRange, svc, ctx)

// Obtain a lease
id, err := lease.Obtain(ctx)

// Close the lease when done
defer lease.Close()
```

#### Key Features

- **Resource Allocation**: Obtain exclusive leases for IDs or IP addresses
- **Automatic Renewal**: Keeps the lease alive with configurable TTLs
- **Failure Recovery**: Attempts to reacquire leases on network failures
- **Event Notifications**: Receive notifications about lease state changes

#### Methods

- `NewLease(range, service, context)`: Creates a new Lease instance
- `Obtain(ctx)`: Obtains an exclusive lease for an ID/IP from the range
- `Wait(ctx)`: Waits for a lease to become available and obtains it
- `Close()`: Releases the lease and stops renewal

### Range

The `Range` class handles parsing and working with ranges of IDs or IP addresses.

```go
// Create an ID range
idRange, err := svcutil.NewIDRange("1-5")

// Create an IP range
ipRange, err := svcutil.NewIPRange("192.168.1.1-192.168.1.10")

// Create an IPv6 range
ipRange, err := svcutil.NewIPRange("2001:db8::1,2001:db8::10")
```

#### Key Features

- **Range Parsing**: Parse ranges specified using hyphen notation (e.g., "1-5") or comma-separated values
- **ID Ranges**: Handle ranges of integer IDs
- **IP Ranges**: Handle ranges of IPv4 addresses (supports single IPs, ranges, and comma-separated notation)
- **IPv6 Support**: Support for comma-separated IPv6 addresses

#### Methods

- `NewIDRange(value)`: Creates a new Range for IDs
- `ParseIDRange(input)`: Parses an ID range string and returns integers
- `NewIPRange(value)`: Creates a new Range for IP addresses
- `ParseIPRange(input)`: Parses an IP range string and returns IP addresses

## Configuration Options

The `svcutil` package uses a functional options pattern to configure services and components. These option functions allow for flexible and readable initialization.

### Service Options

When creating a new `Service` instance, you can provide various options to customize its behavior:

```go
// Create a service with multiple configuration options
svc, err := svcutil.NewService(
    svcutil.Name("auth-service"),
    svcutil.EtcdEndpoints("localhost:2379", "localhost:2380"),
    svcutil.EtcdUsername("etcd-user"),
    svcutil.EtcdPassword("etcd-password"),
    svcutil.DialTimeout(5*time.Second),
    svcutil.LeaseTTL(60),
)
```

#### Available Service Options

- `Name(string)`: Sets the service name (required)
- `Scope(string)`: Sets the service scope
- `EtcdEndpoints(string)`: Specifies etcd server endpoints in comma-separated format
- `EtcdUsername(string)`: Sets the etcd authentication username
- `EtcdPassword(string)`: Sets the etcd authentication password
- `DialTimeout(time.Duration)`: Sets the timeout for connecting to etcd
- `LeaseTTL(int)`: Sets the TTL for etcd leases in seconds
- `ConfigPrefix(string)`: Customizes the prefix for configuration keys
- `LocksPrefix(string)`: Customizes the prefix for lock keys
- `MutexesPrefix(string)`: Customizes the prefix for mutex keys
- `HostsPrefix(string)`: Customizes the prefix for host-specific keys
- `OnEvents(func(EventType, string))`: Sets a callback for service events

### Environment Variables

If options are not explicitly provided, the service will attempt to read these environment variables:

- `ETCD_ADDRESS`: Comma-separated list of etcd endpoints
- `ETCD_USER`: Username for etcd authentication
- `ETCD_PASSWORD`: Password for etcd authentication

### Hostname

Host name could be obtained using `svcutil.Hostname()` function. It is used in service ID generation and in various etcd keys formation.

## Usage Examples

### Configuring a Service

```go
// Create a service with custom options
svc, err := svcutil.NewService(
    svcutil.Name("auth-service"),
    svcutil.EtcdEndpoints("etcd-1:2379,etcd-2:2379"),
    svcutil.LeaseTTL(60),
    svcutil.DialTimeout(3*time.Second),
    svcutil.OnEvents(myEventHandler)
)
if err != nil {
    log.Fatalf("Failed to create service: %v", err)
}
defer svc.Close()
```

### Loading Configuration

```go
// Define a configuration structure
type Config struct {
    Port    int    `json:"port"`
    LogLevel string `json:"log_level"`
}

// Load configuration from etcd
cfg := &Config{}
err := svc.LoadConfig(context.TODO(), cfg)
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### Using Distributed Locks

```go
// Acquire a lock
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := svc.AcquireLock(ctx, "resource-lock")
if err != nil {
    log.Printf("Failed to acquire lock: %v", err)
    return
}

// Do work with exclusive access

// Release the lock when done
svc.ReleaseLock(context.TODO(), "resource-lock")
```

### Allocating IDs

```go
// Create a range and a lease
idRange, _ := svcutil.NewIDRange("1-100")
lease := svcutil.NewLease(idRange, svc, context.Background())

// Obtain an ID
id, err := lease.Obtain(context.Background())
if err != nil {
    log.Fatalf("Failed to obtain ID: %v", err)
}

fmt.Printf("Obtained ID: %s\n", id)
defer lease.Close()
```

### Resource Management

Always close resources when you're done with them:

```go
// Lease cleanup
defer lease.Close()

// Service cleanup
defer svc.Close()
```

## etcd keys

### Configuration

Service configuration:

```
config prefix + service name / value name
/configs/<service>/<value>
```

Scope configuration:

```
config prefix + service scope / value name
/configs/<scope>/<value>
```

Host configuration:

```
hosts prefix + service name / host / value name
/hosts/<service>/<host>/<value>
```

### Locks

Distributed mutexes:

```
locks prefix + service name + mutexes prefix / name
/locks/<service>/mutexes/<name>
```

ID range leases:

```
locks prefix + service name + ids prefix / name
/locks/<service>/ids/<name>
```

IP range leases

```
locks prefix + service name + hosts prefix / host / name
/locks/<service>/hosts/<host>/<name>
```

