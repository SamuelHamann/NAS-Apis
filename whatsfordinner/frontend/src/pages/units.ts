import { listUnits } from "../api/units";
import { renderResourceList } from "./resourceList";

export async function renderUnits(): Promise<void> {
  await renderResourceList({
    title: "Units",
    columns: [
      { header: "ID", value: (u) => String(u.id) },
      { header: "Name", value: (u) => u.name },
    ],
    fetcher: () => listUnits(),
  });
}
