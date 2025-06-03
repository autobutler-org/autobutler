import { computed, readonly } from "vue";
import packageJson from "../package.json";

// Capture build date when module loads
const buildDate = new Date();

// Functional utility to format date as YYYYMMDD
const formatDateAsVersion = (date: Date): string =>
  date.toISOString().slice(0, 10).replace(/-/g, "");

// Functional utility to parse version and replace minor with date
const createVersionWithDate = (version: string, buildDate: Date): string => {
  const [major] = version.split(".");
  const dateMinor = formatDateAsVersion(buildDate);
  return `${major}.${dateMinor}.0`;
};

export const useVersion = () => {
  const version = computed(() =>
    createVersionWithDate(packageJson.version, buildDate),
  );

  const displayVersion = computed(() => `v${version.value}`);

  return {
    version: readonly(version),
    displayVersion: readonly(displayVersion),
    buildDate: readonly(buildDate),
  };
};
