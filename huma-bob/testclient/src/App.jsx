import { useState } from "react";
import LoginPage from "./LoginPage.jsx";
import OAuthPage from "./OAuthPage.jsx";
import JwksPage from "./JwksPage.jsx";
import Callback from "./Callback.jsx";
import "./style.css";

// No router dependency for two tabs + one landing page — plain pathname
// check for /callback, useState for the tab switch.
export default function App() {
  const [tab, setTab] = useState("login");

  if (window.location.pathname === "/callback") {
    return <Callback />;
  }

  return (
    <div className="app">
      <nav>
        <button className={tab === "login" ? "active" : ""} onClick={() => setTab("login")}>
          Login
        </button>
        <button className={tab === "oauth" ? "active" : ""} onClick={() => setTab("oauth")}>
          OAuth
        </button>
        <button className={tab === "jwks" ? "active" : ""} onClick={() => setTab("jwks")}>
          JWKS
        </button>
        <a href="http://localhost:8080/docs" target="_blank" rel="noreferrer">
          API docs
        </a>
      </nav>
      {tab === "login" && <LoginPage />}
      {tab === "oauth" && <OAuthPage />}
      {tab === "jwks" && <JwksPage />}
    </div>
  );
}
