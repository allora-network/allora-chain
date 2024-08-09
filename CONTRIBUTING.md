# Contribution Guidelines

*Borrowed from Osmosis's [contribution guidelines](https://github.com/osmosis-labs/osmosis/blob/main/CONTRIBUTING.md)*

Any contribution that you make to this repository will
be under the Apache 2 License, as dictated by that
[license](http://www.apache.org/licenses/LICENSE-2.0.html):

~~~
5. Submission of Contributions. Unless You explicitly state otherwise,
   any Contribution intentionally submitted for inclusion in the Work
   by You to the Licensor shall be under the terms and conditions of
   this License, without any additional terms or conditions.
   Notwithstanding the above, nothing herein shall supersede or modify
   the terms of any separate license agreement you may have executed
   with Licensor regarding such Contributions.
~~~

Contributors must sign-off each commit by adding a `Signed-off-by: ...`
line to commit messages to certify that they have the right to submit
the code they are contributing to the project according to the
[Developer Certificate of Origin (DCO)](https://developercertificate.org/).


## Checklist for updating global parameters

When updating global parameters, you must update the following files:

For `x/emissions`:

1. `x/emissions/types/params.go`
   1. Set the default values for the new params
   2. Add a new function to return the default values
   3. Add validation for the new params
2. `x/emissions/proto/emissions/v1/params.proto`
   1. Add to the `Params` proto, tracking all global params
3. x/emissions/proto/emissions/v1/tx.proto
   1. Add to the proto of the tx that allows us to set new params
4. `x/emissions/keeper/msgserver/msg_server_params.go`
   1. Add code to the tx that allows us to set new params
5. Update any tests where all params need to be specified
6. Update any external docs here:
   1. https://docs.allora.network/docs/chain-parameters

For `x/mint`:
__TBD__


## Checklist for updating the state machine

When updating the state machine, you must update the following files:

1. `x/emissions/keeper/keeper.go`
2. `x/emissions/keeper/keeper_test.go` as needed
3. `x/emissions/types/keys.go`
4. `x/emissions/keeper/genesis.go`
5. `x/emissions/keeper/genesis_test.go`
6. `x/emissions/proto/emissions/v1/genesis.proto`


## Secondary Limitations To Keep In Mind

#### Network Requests to External Services

It is critical to avoid performing network requests to external services since it is common for services to be unavailable or rate-limit.

Imagine a service that returns exchange rates when clients query its HTTP endpoint. This service might experience downtime or be restricted in some geographical areas.

As a result, nodes may get diverging responses where some get successful responses while others errors, leading to state breakage.

#### Randomness

Randomness cannot be used in the state machine, as the state machine definitionally must be deterministic. Any time you'd consider using it, instead seed a CSPRNG off of some seed.

One thing to note is that in golang, iteration order over maps is non-deterministic, so to be deterministic you must gather the keys, and sort them all prior to iterating over all values.

#### Parallelism and Shared State

Threads and Goroutines might preempt differently in different hardware. Therefore, they should be avoided for the sake of determinism. Additionally, it is hard to predict when the multi-threaded state can be updated.

#### Hardware Errors
This is out of the developer's control but is mentioned for completeness.
