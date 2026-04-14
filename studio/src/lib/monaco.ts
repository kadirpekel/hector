import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";

import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker&inline";
import jsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker&inline";
import cssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker&inline";
import htmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker&inline";
import tsWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker&inline";
import yamlWorker from "../workers/yaml.worker?worker&inline";

// Configure Monaco Environment to use local workers (required for offline/single-file)
self.MonacoEnvironment = {
    getWorker(_, label) {
        if (label === "json") {
            return new jsonWorker();
        }
        if (label === "css" || label === "scss" || label === "less") {
            return new cssWorker();
        }
        if (label === "html" || label === "handlebars" || label === "razor") {
            return new htmlWorker();
        }
        if (label === "typescript" || label === "javascript") {
            return new tsWorker();
        }
        if (label === "yaml") {
            return new yamlWorker();
        }
        return new editorWorker();
    },
};

import { configureMonacoYaml } from "monaco-yaml";

// ... (workers setup remains, but we need to ensure monaco-yaml is activated)

// Store the yaml instance to update options later
let monacoYamlInstance: any = null;

// Initialize loader with local monaco instance
loader.config({ monaco });

const initMonacoYaml = () => {
    if (monacoYamlInstance) return monacoYamlInstance;

    monacoYamlInstance = configureMonacoYaml(monaco, {
        enableSchemaRequest: true,
        schemas: [
            {
                uri: "https://hector.dev/schemas/config.json",
                fileMatch: ["*"],
                schema: {}, // Initial empty schema
            }
        ]
    });
    return monacoYamlInstance;
};

// Configure YAML diagnostics
export const configureYamlSchema = (schema: any) => {
    const instance = initMonacoYaml();
    instance.update({
        validate: true,
        enableSchemaRequest: true,
        hover: true,
        completion: true,
        schemas: [
            {
                uri: "https://hector.dev/schemas/config.json",
                fileMatch: ["*"],
                schema,
            },
        ],
    });
};
