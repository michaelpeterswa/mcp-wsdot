<h1 align="center">
  mcp-wsdot
</h1>
<h2 align="center">
  a model context protocol server for interacting with the washington state department of transportation api
</h2>
<div align="center">

&nbsp;&nbsp;&nbsp;[wsdot library][wsdot-library-link]&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;[model context protocol][mcp-link]|&nbsp;&nbsp;&nbsp;[mcp-go][mcp-go-link]

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