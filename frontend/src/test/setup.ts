import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:avatar-preview"), writable: true });
Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
