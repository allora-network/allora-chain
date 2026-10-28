# v0.18.0 scheduler epoch cutover

The emissions migration converts topic timing from block counts to seconds and enrolls active topics on the scheduler. It uses a fixed assumed block time of 6 seconds.

## Cadence

`EpochLength`, `GroundTruthLag`, and `WorkerSubmissionWindow` are multiplied by 6. The periodic new-epoch task then waits `EpochLength` seconds.

A topic whose epoch was 100 blocks becomes a 600 second cadence.

- On a chain that produces a block every 5 seconds, 600 seconds is 120 blocks. The epoch is about 20% longer than the old 100-block cadence.
- On a chain that produces a block every 7 seconds, 600 seconds is about 86 blocks. The epoch is about 14% shorter than the old 100-block cadence.

That cadence stays in seconds. It does not track the chain's later block time. Window reconstruction uses the same 6 second assumption, so a chain whose real block time is not 6 seconds can open or close a migrated window earlier or later than the old height path.

## Units

Before this migration, `Topic.EpochLength`, `Topic.GroundTruthLag`, `Topic.WorkerSubmissionWindow`, and `Params.MinEpochLength` are block counts.

The migration writes seconds into every topic it converts, including inactive topics. `MinEpochLength` is scaled by the same factor when any topic is converted, and also on a chain that has no topics yet. Topics created after the migration are checked against that seconds floor.

`GetTopic` does not carry a separate unit field. Readers have to know whether the migration has run.

## Topic creators

`CreateNewTopic` rejects an epoch shorter than `MinEpochLength`. After the migration that floor is seconds. A creator who still sends a block count will be rejected when the count is below the scaled floor, or will create a much shorter epoch when the count is above it.

Example: a pre-migration floor of 12 blocks becomes 72 seconds. A request with `epoch_length` 71 fails. A request with `epoch_length` 72 is accepted and means 72 seconds, not 72 blocks.
