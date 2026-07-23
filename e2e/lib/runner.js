// Generic step interpreter for declarative E2E scenarios.
// A "page" (e2e/pages/*.json) is just a name -> CSS selector map plus a path.
// A "scenario" (e2e/scenarios/*.json) is a plain list of steps referencing
// selectors by their friendly name, not raw CSS — so the DOM can change
// without every scenario file changing, and non-JS-writing people can edit
// a scenario without touching k6/Playwright APIs.
import { check } from "k6";

function render(value, vars) {
  if (typeof value !== "string") return value;
  return value.replace(/\{\{(\w+)\}\}/g, (_, key) => (key in vars ? vars[key] : ""));
}

export async function runScenario(page, baseUrl, pageDef, scenario, vars = {}) {
  const allVars = { ts: String(Date.now()), ...vars };
  const results = [];

  for (const step of scenario.steps) {
    const target = step.target ? pageDef.selectors[step.target] : null;
    if (step.target && !target) {
      throw new Error(`Unknown target "${step.target}" — not defined in page "${pageDef.name}"`);
    }

    switch (step.action) {
      case "goto": {
        const path = step.path || pageDef.path || "/";
        await page.goto(baseUrl + path);
        break;
      }
      case "fill": {
        await page.locator(target).fill(render(step.value, allVars));
        break;
      }
      case "click": {
        await page.locator(target).click();
        break;
      }
      case "waitFor": {
        // .first() avoids Playwright "strict mode violation" when the
        // selector legitimately matches more than one element (e.g. a list
        // of existing notes) — we only care that at least one exists.
        await page.locator(target).first().waitFor();
        break;
      }
      case "assertContains": {
        const text = await page.locator(target).first().textContent();
        const expected = render(step.value, allVars);
        const ok = text.includes(expected);
        results.push(
          check(null, { [`${scenario.name}: ${step.target} contains "${expected}"`]: () => ok })
        );
        if (!ok) {
          throw new Error(
            `assertContains failed for "${step.target}": expected to contain "${expected}", got "${text}"`
          );
        }
        break;
      }
      default:
        throw new Error(`Unknown step action "${step.action}"`);
    }
  }

  return results;
}
