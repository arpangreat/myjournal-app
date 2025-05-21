import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "./index.css";
import App from "./App";
import reportWebVitals from "./reportWebVitals";
console.log("MyJournal App is running successfully!");

// Ensure the root element exists in public/index.html
const rootElement = document.getElementById("root");

if (rootElement) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <React.StrictMode>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>
  );

  // Start measuring performance
  reportWebVitals();
} else {
  console.error("Root element not found! Ensure you have <div id='root'></div> in public/index.html.");
}
