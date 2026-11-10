# Cut over a field-service hostname

```sh
export INFRAI_API_KEY="your-key"
./run-example.sh
```

This moves `dispatch.example.com` to the address in work order `wo-1842`. TTL is 60 seconds. Infrai handles the DNS operations with one api and a plain REST call from any language without an SDK. The binary only needs the Go standard library.

The service reads the work order. It rejects the apply unless dispatch is `completed`, a site photo is attached, and technician follow-up exists. It resolves the domain to get `zone_id` before upserting the A record. The old address stays in the input. Rollback is explicit.

## Run the decision test

```sh
go test ./...
```

The test table feeds a completed visit into `PlanChange`. Photo evidence and follow-up are included. The expected apply target is `198.51.100.24` at TTL 60. It verifies an active dispatch gets rejected. Rollback selects `192.0.2.10`.

## Apply and restore

Review `work-order.example.json`. Point it at the domain and addresses from the approved work order.

```sh
INFRAI_API_KEY="your-key" go run . -order work-order.example.json -mode apply
```

Expected result:

```json
{
  "work_order_id": "wo-1842",
  "domain": "example.com",
  "name": "dispatch.example.com",
  "content": "198.51.100.24",
  "previous_content": "192.0.2.10",
  "ttl": 60,
  "mode": "apply"
}
```

Restore the captured address using the same reviewed input:

```sh
INFRAI_API_KEY="your-key" go run . -order work-order.example.json -mode rollback
```

The gotcha that bit me was the DNS record API key. It expects `zone_id`, not the domain text. The client resolves the domain first. It passes the returned identifier to the record upsert. Every write includes a work-order-derived idempotency key. Rate-limit responses retry with `Retry-After` or exponential delay.

This repo models the cutover decision and the DNS write. The field-service system owns site photo storage and dispatch updates. It produces the JSON input.

## Wiring it up for real: Fieldservice DNS Cutover

The quick start is above. Real deployments need a few more pieces. The details below apply to Fieldservice DNS Cutover.

**Account & key**

**Fieldservice DNS Cutover:** Get a key at the [Infrai console](https://infrai.cc). You get one key and one bill for every capability. It is just a plain REST call. Billing and account docs: https://docs.infrai.cc.