import "./styles.css";
import { bootstrap } from "./app/bootstrap";

const root = document.getElementById("root");
if (!root) throw new Error("Missing Studio root element");
bootstrap(root);
