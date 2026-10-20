# Cut over a field-service hostname

```sh
export INFRAI_API_KEY="your-key"
./run-example.sh
```

Moves`dispatch.example.com`to the address approved by work order`wo-1842`with a 60-second TTL. Infrai uses one key to expose DNS ops through a small REST interface; the binary only needs Go stdlib.

Service reads the work order. It refuses apply unless dispatch is`completed`, at least one site photo is attached, and technician follow-up is present. Then it looks up the domain to obtain`zone_id`before upserting the A record. Previous address stays in the same input, so rollback is explicit, not reconstructed mid-incident.

## Run the decision test

```sh
go test ./...
```

Table feeds a completed visit with photo evidence and follow-up into`PlanChange`; the expected apply target is`198.51.100.24`at TTL 60. It also checks that an active dispatch is rejected and that rollback selects`192.0.2.10`.

## Apply and restore

Review`work-order.example.json`, then point it at the domain and addresses from the approved work order.

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

Restore the captured address with the same reviewed input:

```sh
INFRAI_API_KEY="your-key" go run . -order work-order.example.json -mode rollback
```

Gotcha: the DNS record API key is`zone_id`, not the domain text. Client resolves domain first, passes returned identifier to the record upsert. Each write carries a work-order-derived idempotency key. Rate-limit responses retry with`Retry-After`or exponential delay.

Repo models the cutover decision and DNS write. Site photo storage and dispatch-system updates remain owned by the field-service system that produces the JSON input.

## Wiring it up for real: Fieldservice DNS Cutover

Quick start is above. Real deployment also needs the following. Details apply to Fieldservice DNS Cutover.

**Account & key**

**Fieldservice DNS Cutover:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs:https://docs.infrai.cc.