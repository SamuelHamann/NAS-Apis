import { listTags } from "../api/tags";
import { renderResourceList } from "./resourceList";

export async function renderTags(): Promise<void> {
  await renderResourceList({
    title: "Tags",
    columns: [
      { header: "ID", value: (t) => String(t.id) },
      { header: "Name", value: (t) => t.name },
    ],
    fetcher: () => listTags(),
  });
}
