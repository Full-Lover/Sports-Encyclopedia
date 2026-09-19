import { createApp } from "vue";

import App from "./App.vue";
import "./styles.css";

const mountPoint = document.getElementById("map-app");
createApp(App, { snapshotId: mountPoint.dataset.snapshotId }).mount(mountPoint);
