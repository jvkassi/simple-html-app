import { browser } from "k6/browser";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "https://demo-stack.k8s.abj.smile.ci";

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
    await page.goto(BASE_URL + "/");

    await page.locator("#note-body").fill(`Synthetic check ${Date.now()}`);
    await page.locator("#note-form button[type=submit]").click();

    // .first() avoids a Playwright strict-mode violation — the selector
    // legitimately matches every existing note, we just need one to exist.
    await page.locator("#notes-list li").first().waitFor();

    const status = await page.locator("#status").textContent();
    check(status, {
      "status shows note count": (s) => s.includes("note(s)"),
    });
  } finally {
    await page.close();
  }
}
