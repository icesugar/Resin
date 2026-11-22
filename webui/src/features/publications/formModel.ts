import { z } from "zod";
import type {
  SubscriptionPublication,
  SubscriptionPublicationCreateInput,
  SubscriptionPublicationUpdateInput,
} from "./types";

const publicationNameForbiddenChars = ".:|/\\@?#%~";
const publicationNameForbiddenSpacing = " \t\r\n";
const publicationNameReserved = "api";

function containsAny(source: string, chars: string): boolean {
  for (const ch of chars) {
    if (source.includes(ch)) {
      return true;
    }
  }
  return false;
}

export const publicationNameRuleHint =
  "发布名称不能包含 .:|/\\@?#%~、空格、Tab、换行、回车，也不能为保留字。";

export function parseLinesToList(raw: string | undefined, transform?: (value: string) => string): string[] {
  if (!raw) {
    return [];
  }
  const out: string[] = [];
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    out.push(transform ? transform(trimmed) : trimmed);
  }
  return out;
}

export const publicationFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "发布名称不能为空")
    .refine((value) => !containsAny(value, publicationNameForbiddenChars), {
      message: "发布名称不能包含字符 .:|/\\@?#%~",
    })
    .refine((value) => !containsAny(value, publicationNameForbiddenSpacing), {
      message: "发布名称不能包含空格、Tab、换行、回车",
    })
    .refine((value) => value.toLowerCase() !== publicationNameReserved, {
      message: "发布名称不能为保留字",
    }),
  regex_filters_text: z.string().optional(),
  region_filters_text: z.string().optional(),
  healthy_only: z.boolean(),
  enabled: z.boolean(),
});

export type PublicationFormValues = z.infer<typeof publicationFormSchema>;

export const defaultPublicationFormValues: PublicationFormValues = {
  name: "",
  regex_filters_text: "",
  region_filters_text: "",
  healthy_only: false,
  enabled: true,
};

export function publicationToFormValues(publication: SubscriptionPublication): PublicationFormValues {
  return {
    name: publication.name,
    regex_filters_text: publication.regex_filters.join("\n"),
    region_filters_text: publication.region_filters.join("\n"),
    healthy_only: publication.healthy_only,
    enabled: publication.enabled,
  };
}

export function toCreateInput(values: PublicationFormValues): SubscriptionPublicationCreateInput {
  return {
    name: values.name.trim(),
    regex_filters: parseLinesToList(values.regex_filters_text),
    region_filters: parseLinesToList(values.region_filters_text, (value) => value.toLowerCase()),
    healthy_only: values.healthy_only,
    enabled: values.enabled,
  };
}

export function toUpdateInput(values: PublicationFormValues): SubscriptionPublicationUpdateInput {
  return {
    name: values.name.trim(),
    regex_filters: parseLinesToList(values.regex_filters_text),
    region_filters: parseLinesToList(values.region_filters_text, (value) => value.toLowerCase()),
    healthy_only: values.healthy_only,
    enabled: values.enabled,
  };
}
