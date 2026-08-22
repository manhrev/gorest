import { useState } from "react";
import { apiCall } from "./api.js";
import Exchange from "./Exchange.jsx";

export default function LoginPage() {
  const [username, setUsername] = useState("alice");
  const [password, setPassword] = useState("hunter2");
  const [loginResult, setLoginResult] = useState(null);
  const [otherResult, setOtherResult] = useState(null);

  async function login() {
    const r = await apiCall("POST", "/auth/login", {}, { username, password });
    setLoginResult(r);
    if (r.response.body?.data) {
      localStorage.setItem("accessToken", r.response.body.data.accessToken);
      localStorage.setItem("refreshToken", r.response.body.data.refreshToken);
    }
  }

  async function checkAuth() {
    const token = localStorage.getItem("accessToken") || "";
    setOtherResult(await apiCall("GET", "/auth/check", { Authorization: "Bearer " + token }));
  }

  async function logout() {
    const token = localStorage.getItem("accessToken") || "";
    setOtherResult(await apiCall("POST", "/auth/logout", { Authorization: "Bearer " + token }, {}));
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
  }

  return (
    <div>
      <h1>Login</h1>
      <p className="hint">
        Stub creds (internal/server/auth_stub.go): <code>alice</code> / <code>hunter2</code>.
      </p>

      <fieldset>
        <label>Username</label>
        <input value={username} onChange={(e) => setUsername(e.target.value)} />
        <label>Password</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        <button onClick={login}>Login</button>
      </fieldset>
      <Exchange result={loginResult} />

      <fieldset>
        <p className="hint">Uses the access token currently in localStorage (set by Login above).</p>
        <button onClick={checkAuth}>Check Auth</button>
        <button onClick={logout}>Logout</button>
      </fieldset>
      <Exchange result={otherResult} />
    </div>
  );
}
