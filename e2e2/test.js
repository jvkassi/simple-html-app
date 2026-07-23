import { browser } from "k6/browser";
import { runScenario } from "../e2e/lib/runner.js";

const BASE_URL = __ENV.BASE_URL || "https://demo-stack.k8s.abj.smile.ci";
const SCENARIO_FILE = __ENV.SCENARIO || "./scenarios/create-and-view-note.json";

const scenario = JSON.parse(open(SCENARIO_FILE));
const pageDef = JSON.parse(open(`./pages/${scenario.page}.json`));

export const options = {
  scenarios: {
    e2e: {
      executor: "shared-iterations",
      vus: 1,
      iterations: 1,
      options: { browser: { type: "chromium" } },
    },
  },
};

export default async function () {
  const page = await browser.newPage();
  try {
    await runScenario(page, BASE_URL, pageDef, scenario);
  } finally {
    await page.close();
  }
}
