# AGENT.md — compnet-socket-labs (H01: Introduction to Socket Programming)

Guidance for AI coding agents working in this repository. Read it fully before changing any code. This is a university lab (Computer Networks). The `project/` code is **graded by automated unit tests**, so contract compliance matters more than style or cleverness.

## 1. Project Summary

Implement a simple **PIDS (Passenger Information Display System)** for LRT Jakarta as an application-layer protocol running **on top of QUIC** (Go, `quic-go`).

| Role | Directory | Meaning | Runs on |
|---|---|---|---|
| **Publisher** (QUIC client) | `project/publisher` | "Node Kendali Stasiun" (station control). Sends train data packets. | VM1 |
| **Subscriber** (QUIC server) | `project/subscriber` | "Node Layar PIDS" (PIDS screen). Listens, decodes, handles, prints, ACKs. | VM2 |

Language: **Go**. Module: `compnet-socket-labs`. Build tool: **Mage**.

## 2. Student Configuration (fixed values)

| Setting | Value |
|---|---|
| NPM | `2406396584` |
| **ALPN** | `lrt-jakarta-2406396584` |
| **Port** | `6584` (last 4 digits of NPM) |
| Publisher host | VM1 |
| Subscriber host | VM2 |
| IPs | **Provided via environment variables.** Never hardcode VM IPs. Use each VM's *private* IP. |

Env vars (confirmed in the skeleton's `ResolveConfig` / `ResolveALPN`; do not rename): `SERVER_ADDR` (IP), `PORT`, `ALPN`. The autograder checks IP, port, and ALPN attributes, so the defaults above must be correct in **both** `publisher/main.go` and `subscriber/main.go`.

- Subscriber (VM2): listen on the address from env (its own private IP, or all interfaces if the skeleton allows).
- Publisher (VM1): dial VM2's private IP from env, port `6584`.

## 3. Repository Layout

```
compnet-socket-labs/
├── project/                  # ← THE GRADED WORK
│   ├── publisher/main.go     # client side
│   ├── subscriber/main.go    # server side (contains Handler)
│   └── utils/
│       ├── utils.go          # packet types + Encoder/Decoder
│       └── cert.go           # TLS self-signed cert helper
├── samples/                  # reference only; copy/adapt, don't modify
│   ├── udp/ tcp/ quic/ codec/
├── mage.go                   # Mage launcher
└── magefile.go               # Mage targets (sanity, package, ...)
```

Every `*.go` file has a companion `contract_test.go` checking the teaching team's contract. **Never delete or weaken these tests.**

> The assignment text says `Handler` is in `subscriber.go`, but the actual skeleton has it in `project/subscriber/main.go`. Keep it there with its exact signature (see section 6).

## 4. Commands

Run from the repository **root**.

```bash
go mod download -x              # install deps (quic-go etc.), once
go run ./project/subscriber     # VM2: start PIDS screen node FIRST
go run ./project/publisher      # VM1: then station control node

go run mage.go sanity           # contract check, must show NO fail
go run mage.go package 2406396584   # build submission ZIP (only when user asks)
```

Optional QUIC debug flags (as in samples): `-keylog` (TLS keys → `logs/ssl-key.log`, for Wireshark), `-qlog`. The quic-go warning `failed to sufficiently increase receive buffer size...` is harmless.

## 5. HARD RULES (do not violate)

1. **Follow the provided Publisher/Subscriber templates.** Part of the grade is automated.
2. **Use QUIC.** Not using QUIC loses points.
3. **Structs:** field **names must not change**; only field *types* may change. **Do not add or remove fields.** Use a separate DTO if extra data is needed.
4. **Do not change the signature** (name, params, return types) of any provided function.
5. **Encoder/Decoder live in `utils.go`** and each does *only* its job: no printing, no business logic, no network I/O.
6. **`Handler`** (subscriber) takes an already-decoded object and returns the string to print. Nothing else.
7. **No external modules** other than `quic-go`, `mage`, and modules already in `compnet-socket-labs`. No new `go get`.
8. **Never leave empty/stub methods.** `mage package` refuses to build when there are issues, and an unrunnable submission scores zero. Code must always compile and pass `sanity`.
9. Do not restructure directories.

## 6. Provided Skeleton (`project/utils/utils.go`)

This is the current contract. Do not rename anything.

```go
package utils

// Component data types may be modified, but field names must not be changed
type LRTJPIDSPacketFixed struct {
	TransactionId          int
	IsAck                  int
	IsNewTrain             int
	IsUpdateTrain          int
	IsDeleteTrain          int
	IsTrainArriving        int
	IsTrainDeparting       int
	TrainNumber            int
	EstimatedArrivalHour   int
	EstimatedArrivalMinute int
	DestinationLength      int
}

type LRTJPIDSPacket struct {
	LRTJPIDSPacketFixed
	Destination string
}

func Encoder(packet LRTJPIDSPacket) []byte { return nil }          // TODO
func Decoder(rawMessage []byte) LRTJPIDSPacket { return LRTJPIDSPacket{} } // TODO
```

Observations that drive the implementation:

- Function names are **`Encoder`** and **`Decoder`** (not Encode/Decode). The ID field is **`TransactionId`** (lowercase `d`).
- Flags are `int` fields holding **0 or 1**, not `bool`.
- `Encoder` returns only `[]byte` (no `error`); `Decoder` returns only the packet. Do not change this. Handle malformed input defensively (e.g., short buffer → return a zero-value packet, never panic).
- `LRTJPIDSPacket` **embeds** `LRTJPIDSPacketFixed`, so fields are accessed as `p.TransactionId`, `p.Destination`.
- **Keep field types as `int`** unless there is a strong reason to change. The autograder likely builds and compares packets using these types, and changing them risks compile failures in hidden tests.
- `binary.Write` on a struct of plain `int` fails (`int` is not fixed-size) and the six flags need packing into one byte anyway. So **encode/decode manually** (`binary.BigEndian.PutUint16`, shifts, masks), or convert to a private fixed-width wire struct *inside* `utils.go` and write that. Do not add fields to the exported structs.

### Provided skeleton: `project/subscriber/main.go`

```go
package main

import (
	"os"
	"compnet-socket-labs/project/utils"
)

var (
	DefaultServerIP   = "127.0.0.1"
	DefaultServerPort = "6584"
	ServerType        = "udp4"
	BufferSize        = 2048
	AppLayerProto     = "lrt-jakarta-2406396584" 
)

func ResolveConfig() (string, string) { /* SERVER_ADDR, PORT with defaults */ }
func ResolveALPN() string             { /* ALPN with default */ }

func Handler(packet utils.LRTJPIDSPacket) string { return "" } // TODO

func main() { _ = Handler(utils.LRTJPIDSPacket{}) }             // TODO: real server
```

Notes:

- **Do not change** the signature `Handler(packet utils.LRTJPIDSPacket) string`, nor `ResolveConfig() (string, string)` / `ResolveALPN() string`. Reuse them from `main()`.
- `main()` is only a placeholder. It must be replaced with the real QUIC listener loop (section 9). The `_ = Handler(...)` line exists to keep the stub compiling; remove it once `Handler` is actually used.
- `ServerType = "udp4"` is correct for QUIC (it runs on a UDP socket). Keep it.
- `BufferSize = 2048` is the app-level read buffer; keep it unless there is a reason.
- `Handler` is pure: no printing, no network I/O. `main`/the stream handler prints the returned string.

### Provided skeleton: `project/publisher/main.go`

```go
package main

import (
	"os"
	"compnet-socket-labs/project/utils"
)

var (
	DefaultServerIP   = "127.0.0.1"
	DefaultServerPort = "6584"
	BufferSize        = 2048
	AppLayerProto     = "lrt-jakarta-2406396584"
)

func ResolveConfig() (string, string) { /* SERVER_ADDR, PORT with defaults */ }
func ResolveALPN() string             { /* ALPN with default */ }

func main() { _ = utils.LRTJPIDSPacket{} } // TODO: real client
```

Notes:

- Same `ResolveConfig` / `ResolveALPN` and same two defaults to change as the subscriber.
- Differences from the subscriber: there is **no `ServerType` variable** and **no `Handler`**. Do not add a `Handler` here. If the network type string `"udp4"` is needed for address resolution, use it inline (or a new package-level var) without altering the existing ones.
- Here `SERVER_ADDR` is the **target**: VM2's private IP (the subscriber). `main()` is a placeholder to be replaced with the real QUIC dial → send Packet A → read ACK → send Packet B → read ACK → exit flow (section 9). Remove the `_ = utils.LRTJPIDSPacket{}` line once the packets are actually built.
- Packets A and B are hardcoded here (section 8), built as `utils.LRTJPIDSPacket` values with `DestinationLength` set to `len(Destination)`, and `TransactionId` unique per packet (e.g. 1 and 2).

## 7. Protocol: `LRTJPIDSPacket` wire format

Same format for requests (publisher → subscriber) and ACKs (subscriber → publisher). All multi-byte numbers are **Big-Endian**.

| # | Field (struct name) | Wire size | Notes |
|---|---|---|---|
| 1 | `TransactionId` | 16 bit (2 bytes) | ACK **must reuse** the request's ID. |
| 2 | Flags byte | 8 bit (1 byte) | See below. |
| 3 | `TrainNumber` | 16 bit (2 bytes) | e.g. 1002. |
| 4 | `EstimatedArrivalHour` | 8 bit | number |
| 5 | `EstimatedArrivalMinute` | 8 bit | number |
| 6 | `DestinationLength` | 8 bit | byte length of `Destination` |
| 7 | `Destination` | `DestinationLength` bytes | UTF-8, e.g. `Manggarai` (9), `Kelapa Gading` (13) |

Total = 7 fixed bytes + `DestinationLength`.

### Flags byte (bit 7 = leftmost/MSB)

| Bit | Struct field | Set when |
|---|---|---|
| 7 | `IsAck` | Reply from PIDS screen node |
| 6 | `IsNewTrain` | Add a new train |
| 5 | `IsUpdateTrain` | Modify existing train |
| 4 | `IsDeleteTrain` | Remove a train (e.g. cancellation) |
| 3 | `IsTrainArriving` | Train about to enter station |
| 2 | `IsTrainDeparting` | Train about to depart |
| 1–0 | unused | not in the struct; always `0` on the wire |

Pack/unpack sketch:

```go
flags := byte(p.IsAck<<7 | p.IsNewTrain<<6 | p.IsUpdateTrain<<5 |
              p.IsDeleteTrain<<4 | p.IsTrainArriving<<3 | p.IsTrainDeparting<<2)

p.IsAck           = int(flags >> 7 & 1)
p.IsNewTrain      = int(flags >> 6 & 1)
p.IsUpdateTrain   = int(flags >> 5 & 1)
p.IsDeleteTrain   = int(flags >> 4 & 1)
p.IsTrainArriving = int(flags >> 3 & 1)
p.IsTrainDeparting= int(flags >> 2 & 1)
```

Implementation guidance:

- **Encoder:** write `TransactionId` (uint16 BE), flags, `TrainNumber` (uint16 BE), hour, minute, then the destination length byte, then destination bytes. Derive the length byte from `len(packet.Destination)` so it can never disagree with the payload.
- **Decoder:** read the 7 fixed bytes, then read exactly `DestinationLength` bytes for `Destination`; **set `DestinationLength` on the returned packet**. Guard against `len(rawMessage) < 7` and truncated destinations.
- Round-trip must hold: `Decoder(Encoder(p))` equals `p` (given `p.DestinationLength == len(p.Destination)`).
- Reference technique: `samples/codec/main.go`.

## 8. Expected Behavior (end-to-end)

1. **Subscriber** listens continuously for QUIC connections until killed (not just for the two demo packets).
2. **Publisher** hardcodes two packets:
   - **Packet A:** `IsNewTrain=1`, train **1002**, destination **Manggarai**, ETA **05:34**
   - **Packet B:** `IsUpdateTrain=1`, train **1002**, destination **Velodrome**, ETA **05:35**
3. For each packet (A then B):
   1. Publisher `Encoder` → bytes → send over a QUIC stream.
   2. Subscriber `Decoder` → packet.
   3. Subscriber calls `Handler`; prints the returned string to stdout **identically**.
   4. Subscriber builds the **ACK**: same packet (same `TransactionId` and data) with `IsAck=1`, `Encoder` → send back.
   5. Publisher `Decoder` → ACK. Nothing else required (logging is fine).
4. After both packets, publisher exits.

### `Handler` output

| Condition | Returned string | Example |
|---|---|---|
| `IsNewTrain` set | `ADD \| <TrainNumber> \| <Destination> \| <HH:MM>` | `ADD | 1002 | Manggarai | 05:34` |
| `IsUpdateTrain` set | `UPD \| <TrainNumber> \| <Destination> \| <HH:MM>` | `UPD | 1002 | Velodrome | 05:35` |
| otherwise | not handled; ignore | |

(Backslashes above only escape the pipe in Markdown. The real separator is `" | "`.)

- ETA is **zero-padded**: `fmt.Sprintf("%02d:%02d", h, m)`.
- **No state storage**, no duplicate or existence checks.
- Extra debug output is allowed, but the handler line must be printed unchanged.
- Flags are `int`, so test with `== 1` (or `!= 0`).

## 9. Go / QUIC Implementation Notes

Mirror `samples/quic/{server,client}/main.go`.

**Subscriber:**
- `net.ResolveUDPAddr("udp4", …)` → `net.ListenUDP` → `quic.Listen(udpConn, tlsConfig, quicConfig)`.
- `tls.Config{Certificates: utils.GenerateTLSSelfSignedCertificates(), NextProtos: []string{alpn}}`.
- Loop `listener.Accept(ctx)` → `go connectionHandler(conn)` → `conn.AcceptStream(ctx)` → `go streamHandler(...)`.
- Treat `io.EOF` as a normal close; `defer stream.Close()`.

**Publisher:**
- `tls.Config{InsecureSkipVerify: true, NextProtos: []string{alpn}}` (self-signed, lab only).
- `quic.DialAddr(ctx, net.JoinHostPort(ip, port), tlsConfig, quicConfig)`.
- `defer conn.CloseWithError(0x0, "No Error")`.
- `conn.OpenStreamSync(ctx)` → write encoded packet → read ACK → decode.

**Pitfalls:**
- ALPN mismatch (`lrt-jakarta-2406396584` on both sides) fails the handshake. Check this first on connection errors.
- Server must close the stream after replying or the client read can hang.
- One `Read` may not return a whole packet. Prefer `io.ReadFull` using the known 7-byte header + `DestinationLength`.
- `DestinationLength` is 8 bits, so destination max is 255 bytes.
- Between VMs, ensure UDP port `6584` is reachable (QUIC runs over UDP, not TCP).

## 10. Workflow for Agents

1. Read the skeleton in `project/` and the `contract_test.go` files first.
2. Study `samples/quic` and `samples/codec`.
3. Implement in order: `Encoder`/`Decoder` → `Handler` → subscriber networking → publisher networking.
4. After each step: `go build ./...` and `go run mage.go sanity` (zero fails).
5. Test locally with two terminals (subscriber first). Verify:

```
ADD | 1002 | Manggarai | 05:34
UPD | 1002 | Velodrome | 05:35
```

6. Don't run `mage package` unless the user asks. Don't edit `samples/` unless asked.
7. For VM testing, supply IPs through env vars, e.g. `SERVER_ADDR=<vm2-private-ip> go run ./project/publisher`.

## 11. Grading (context)

| Weight | Item |
|---|---|
| 20 pts | Self-test evidence: one screenshot with both SSH windows (VM1 publisher, VM2 subscriber), showing shell prompts, run commands, and output. |
| 30 pts | Autograder: (1) QUIC socket choice, (2) IP / port / ALPN, (3) Encoder, (4) Decoder, (5) Handler. |

## 12. Do / Don't

**Do**
- Keep `Encoder`/`Decoder` symmetric, pure, and panic-free.
- Reuse the same `TransactionId` in the ACK.
- Use Big-Endian everywhere; zero-pad ETA.
- Keep the code compiling at all times.

**Don't**
- Don't rename `Encoder`/`Decoder`/`TransactionId` or any field, or add/remove fields.
- Don't hardcode VM IPs.
- Don't put I/O, printing, or handling logic in Encoder/Decoder.
- Don't add third-party dependencies or use TCP/plain UDP for the project.
- Don't store train state or add validation the spec says to skip.
- Don't modify or delete `contract_test.go` files.
- Don't commit `logs/`, `ssl-key.log`, or private keys.