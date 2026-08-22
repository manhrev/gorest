// splitUrl: separates a URL's path from its query params (decoded), so the
// request/response panels can show "GET /oauth/authorize" plus a clean
// key/value list instead of one long percent-encoded line.
function splitUrl(url) {
  try {
    const u = new URL(url);
    const params = [...u.searchParams.entries()];
    return { path: u.origin + u.pathname, params };
  } catch {
    return { path: url, params: [] };
  }
}

function ParamsTable({ params }) {
  if (params.length === 0) return null;

  return (
    <table className="params">
      <tbody>
        {params.map(([k, v]) => (
          <tr key={k}>
            <td className="param-key">{k}</td>
            <td className="param-value">{v}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// Exchange: renders one apiCall() result as a request panel + response
// panel — the "show request, response" requirement, one component shared
// by every button on every page.
export default function Exchange({ result }) {
  if (!result) return null;
  const { request, response, ok } = result;

  const reqUrl = splitUrl(request.url);
  const resUrl = response.url ? splitUrl(response.url) : null;

  return (
    <div className="exchange">
      <div className="panel req">
        <h3>Request</h3>
        <pre>
          {request.method} {reqUrl.path}
          {"\n"}
          {Object.entries(request.headers)
            .map(([k, v]) => `${k}: ${v}`)
            .join("\n")}
        </pre>
        <ParamsTable params={reqUrl.params} />
        {request.body !== undefined && <pre>{JSON.stringify(request.body, null, 2)}</pre>}
      </div>
      <div className={"panel res" + (ok === false ? " err" : "")}>
        <h3>
          Response — {response.status} {response.statusText}
          {response.redirected ? " (after following a redirect — see note below)" : ""}
        </h3>
        {response.error ? (
          <pre>network error: {response.error}</pre>
        ) : (
          <>
            <pre>
              final URL: {resUrl.path}
              {"\n"}
              {Object.entries(response.headers)
                .map(([k, v]) => `${k}: ${v}`)
                .join("\n")}
            </pre>
            <ParamsTable params={resUrl.params} />
            <pre>{typeof response.body === "string" ? response.body : JSON.stringify(response.body, null, 2)}</pre>
          </>
        )}
      </div>
    </div>
  );
}
