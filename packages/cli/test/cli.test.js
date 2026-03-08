const test = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

test("cli reports the published version", () => {
  const cliPath = path.join(__dirname, "..", "dist", "index.js");
  const output = execFileSync(process.execPath, [cliPath, "--version"], {
    encoding: "utf-8",
  });

  assert.match(output, /practor v0\.3\.0/);
});
