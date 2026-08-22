// Exchange: renders one apiCall() result as a request panel + response
// panel — the "show request, response" requirement, one component shared
// by every button on every page.
export default function Exchange({ result }) {
  if (!result) return null;
  const { request, response, ok } = result;

  return (
    <div className="exchange">
      <div className="panel req">
        <h3>Request</h3>
        <pre>
          {request.method} {request.url}
          {"\n"}
          {Object.entries(request.headers)
            .map(([k, v]) => `${k}: ${v}`)
            .join("\n")}
          {request.body !== undefined ? "\n\n" + JSON.stringify(request.body, null, 2) : ""}
        </pre>
      </div>
      <div className={"panel res" + (ok === false ? " err" : "")}>
        <h3>
          Response — {response.status} {response.statusText}
          {response.redirected ? " (after following a redirect — see note below)" : ""}
        </h3>
        <pre>
          {response.error
            ? "network error: " + response.error
            : `final URL: ${response.url}\n` +
              Object.entries(response.headers)
                .map(([k, v]) => `${k}: ${v}`)
                .join("\n") +
              "\n\n" +
              (typeof response.body === "string" ? response.body : JSON.stringify(response.body, null, 2))}
        </pre>
      </div>
    </div>
  );
}
