import "./index.css";

import ReactDOM from "react-dom/client";
import { getCurrentWindow } from "@tauri-apps/api/window";

import App from "./App";
import MiniOverlay from "./components/MiniOverlay";

function currentLabel() {
  try {
    return getCurrentWindow().label;
  } catch {
    return "main";
  }
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  currentLabel() === "mini" ? <MiniOverlay /> : <App />
);
