import fs from"fs/promises";
import path from "path";
import FlexSearch from "flexsearch";

const searchFileName = "search.json";
const indexDirName = "search";

async function createIndex(inputDir, outputDir) {
    let searchData;

    try {
        searchData = JSON.parse(await fs.readFile(path.join(inputDir, searchFileName)));
    } catch {
        return
    }

    const index = new FlexSearch.Document({
        preset: "memory",
        document: {
            id: "id",
            store: ["id", "page", "timestamp", "user", "title", "body"],
            index: ["user", "flair", "year", "title", "body"],
        },
    });

    await Promise.all(searchData.map((fields) => index.addAsync(fields)));

    const indexDir = path.join(outputDir, indexDirName);
    await fs.mkdir(indexDir, { recursive: true });

    await index.export(async (key, data) => {
        if (data !== undefined) {
            await fs.writeFile(path.join(indexDir, key), data);
        }
    })
}

export default createIndex;