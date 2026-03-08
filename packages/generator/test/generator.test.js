const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const { generate } = require("../dist");

test("generate writes a typed client file", () => {
  const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "practor-generator-"));

  try {
    generate(
      {
        datasources: [],
        generators: [],
        enums: [],
        models: [
          {
            name: "User",
            fields: [
              {
                name: "id",
                type: { name: "Int", isScalar: true, isEnum: false, isModel: false },
                isList: false,
                isOptional: false,
                attributes: [{ name: "id" }],
              },
              {
                name: "email",
                type: { name: "String", isScalar: true, isEnum: false, isModel: false },
                isList: false,
                isOptional: false,
                attributes: [{ name: "unique" }],
              },
            ],
          },
        ],
      },
      outputDir,
    );

    const generated = fs.readFileSync(path.join(outputDir, "index.ts"), "utf-8");
    assert.match(generated, /export interface User/);
    assert.match(generated, /email: string;/);
    assert.match(generated, /export class PractorClient/);
  } finally {
    fs.rmSync(outputDir, { recursive: true, force: true });
  }
});
