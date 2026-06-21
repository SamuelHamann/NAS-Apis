import { listLocations } from "../api/locations";
import { renderResourceList } from "./resourceList";

export async function renderLocations(): Promise<void> {
  await renderResourceList({
    title: "Food locations",
    columns: [
      { header: "ID", value: (l) => String(l.id) },
      { header: "Name", value: (l) => l.name },
    ],
    fetcher: () => listLocations(),
  });
}
