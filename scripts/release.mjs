import { appendFileSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

function matches(pattern, value) {
  return typeof value === "string" && !/[\r\n]/.test(value) && pattern.test(value);
}

export function releaseMetadata({ refType, tag, version, repository, username, image }) {
  if (refType !== "tag" || !matches(/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/, tag)) {
    throw new Error("Release requires a stable v<major>.<minor>.<patch> tag.");
  }
  if (version.trim() !== tag) throw new Error("The release tag must match VERSION.");
  if (!matches(/^[a-z\d_.-]+\/[a-z\d_.-]+$/i, repository)) {
    throw new Error("GITHUB_REPOSITORY must identify the source owner/repository.");
  }
  if (!matches(/^[a-z\d][a-z\d_-]*$/, username)) {
    throw new Error("Set the DOCKERHUB_USERNAME repository variable to the Docker Hub login name.");
  }
  const component = "[a-z\\d]+(?:(?:[._]|__|-+)[a-z\\d]+)*";
  if (!matches(new RegExp(`^${component}/${component}$`), image) || image.length > 255) {
    throw new Error("Set the DOCKERHUB_IMAGE repository variable to lowercase namespace/image, without a registry or tag.");
  }
  const ghcr = `ghcr.io/${repository.toLowerCase()}`;
  const registries = [ghcr, `docker.io/${image}`];
  return { version: tag, tags: registries.flatMap((name) => [`${name}:${tag}`, `${name}:latest`]) };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const metadata = releaseMetadata({
      refType: process.env.GITHUB_REF_TYPE,
      tag: process.env.GITHUB_REF_NAME,
      version: readFileSync(new URL("../VERSION", import.meta.url), "utf8"),
      repository: process.env.GITHUB_REPOSITORY,
      username: process.env.DOCKERHUB_USERNAME,
      image: process.env.DOCKERHUB_IMAGE,
    });
    if (!process.env.GITHUB_OUTPUT) throw new Error("GITHUB_OUTPUT is required to return release metadata.");
    appendFileSync(process.env.GITHUB_OUTPUT, `version=${metadata.version}\ntags<<HARBOR_TAGS\n${metadata.tags.join("\n")}\nHARBOR_TAGS\n`);
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
