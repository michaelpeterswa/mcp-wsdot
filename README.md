<h2 align="center">
<img src=".github/images/mcp-wsdot.png" alt="mcp-wsdot logo" width="500">
</h2>
<h2 align="center">
  a model context protocol server for interacting with the washington state department of transportation api
</h2>
<div align="center">

&nbsp;&nbsp;&nbsp;[wsdot library][wsdot-library-link]&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;[model context protocol][mcp-link]&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;[mcp-go][mcp-go-link]

[![Made With Go][made-with-go-badge]][for-the-badge-link] [![Made With MCP][made-with-mcp-badge]][for-the-badge-link]

</div>

---
## Development
### Prerequisites
* For the best development experience, install [bun](https://bun.sh) so the MCP Inspector can be started (using `make mcp`)

### Building
* Please use `docker compose` to run the application locally.
  * Docker Compose Instructions:
    * `docker compose up -d --build && docker compose logs -f -t`

Run with docker-compose and check Grafana at `http://localhost:3000` for metrics/traces.

## Transports

This application supports both SSE and STDIO transports using the environment variable `TRANSPORT`

## Current Tools

| tool                            | description                                              |
|---------------------------------|----------------------------------------------------------|
| `get_current_time`                | gets the current time in the PDT/PST timezone            |
| `get_route_schedules`             | gets the names and route id's for available ferry routes |
| `get_schedules_today_by_route_id` | gets the sailing times for a route by route id           |

## MacOS Binary Notes

If you download the binary from the releases page on MacOS, you will need to exempt it from the quarantine security warnings. To do that, run the following command:

```
xattr -rd com.apple.quarantine <path-to-binary>
```

## Example Question

> What is the next three sailings for the Seattle to Bremerton Ferry Route? And the latest I can return tonight?

__Asked to Claude Desktop 3.7 Sonnet__ 

>I'll help you find the next three sailings for the Seattle to Bremerton route and the latest return option tonight. Let me search for the current ferry schedule.

>Now that I have the route information, let me check the specific schedule for the Seattle to Bremerton ferry route:

>Based on the schedule information, here are the details you requested:

>### Next Three Sailings from Seattle to Bremerton:
>1. 12:20 PM today (April 20, 2025) - Vessel: Walla Walla
>2. 2:50 PM today (April 20, 2025) - Vessel: Walla Walla
>3. 5:30 PM today (April 20, 2025) - Vessel: Walla Walla

>### Latest Return from Bremerton to Seattle Tonight:
>The latest sailing from Bremerton to Seattle tonight is at 11:40 PM (23:40) - Vessel: Walla Walla

>All sailings are operated by the vessel Walla Walla, which is handicap accessible. This schedule >is part of the Spring 2025 service which runs from March 23 to June 14, 2025.

<!--

Reference Variables

-->

<!-- Badges -->
[made-with-go-badge]: .github/images/made-with-go.svg
[made-with-mcp-badge]: .github/images/made-with-mcp.svg

<!-- Links -->
[blank-reference-link]: #
[for-the-badge-link]: https://forthebadge.com
[wsdot-library-link]: https://github.com/alpineworks/wsdot
[mcp-link]: https://modelcontextprotocol.io/introduction
[mcp-go-link]: https://github.com/mark3labs/mcp-go