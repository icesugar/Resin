import { apiRequest } from "../../lib/api-client";
import type {
  PageResponse,
  SubscriptionPublication,
  SubscriptionPublicationCreateInput,
  SubscriptionPublicationPreview,
  SubscriptionPublicationUpdateInput,
} from "./types";

const basePath = "/api/v1/subscription-publications";

type ApiPublication = Partial<SubscriptionPublication>;

function normalizePublication(raw: ApiPublication): SubscriptionPublication {
  return {
    id: typeof raw.id === "string" ? raw.id : "",
    name: typeof raw.name === "string" ? raw.name : "",
    enabled: typeof raw.enabled === "boolean" ? raw.enabled : true,
    regex_filters: Array.isArray(raw.regex_filters) ? raw.regex_filters : [],
    region_filters: Array.isArray(raw.region_filters) ? raw.region_filters : [],
    healthy_only: typeof raw.healthy_only === "boolean" ? raw.healthy_only : false,
    matching_node_count: typeof raw.matching_node_count === "number" ? raw.matching_node_count : 0,
    node_count: typeof raw.node_count === "number" ? raw.node_count : 0,
    skipped_node_count: typeof raw.skipped_node_count === "number" ? raw.skipped_node_count : 0,
    healthy_node_count: typeof raw.healthy_node_count === "number" ? raw.healthy_node_count : 0,
    subscription_path: typeof raw.subscription_path === "string" ? raw.subscription_path : "",
    provider_path: typeof raw.provider_path === "string" ? raw.provider_path : "",
    singbox_path: typeof raw.singbox_path === "string" ? raw.singbox_path : "",
    created_at: typeof raw.created_at === "string" ? raw.created_at : "",
    updated_at: typeof raw.updated_at === "string" ? raw.updated_at : "",
  };
}

export type ListSubscriptionPublicationsInput = {
  limit?: number;
  offset?: number;
  keyword?: string;
  enabled?: boolean;
};

export async function listSubscriptionPublications(
  input: ListSubscriptionPublicationsInput = {},
): Promise<PageResponse<SubscriptionPublication>> {
  const query = new URLSearchParams({
    limit: String(input.limit ?? 20),
    offset: String(input.offset ?? 0),
    sort_by: "name",
    sort_order: "asc",
  });
  if (input.enabled !== undefined) {
    query.set("enabled", String(input.enabled));
  }
  const keyword = input.keyword?.trim();
  if (keyword) {
    query.set("keyword", keyword);
  }

  const data = await apiRequest<PageResponse<ApiPublication>>(`${basePath}?${query.toString()}`);
  return {
    ...data,
    items: data.items.map(normalizePublication),
  };
}

export async function createSubscriptionPublication(
  input: SubscriptionPublicationCreateInput,
): Promise<SubscriptionPublication> {
  const data = await apiRequest<ApiPublication>(basePath, {
    method: "POST",
    body: input,
  });
  return normalizePublication(data);
}

export async function updateSubscriptionPublication(
  id: string,
  input: SubscriptionPublicationUpdateInput,
): Promise<SubscriptionPublication> {
  const data = await apiRequest<ApiPublication>(`${basePath}/${id}`, {
    method: "PATCH",
    body: input,
  });
  return normalizePublication(data);
}

export async function deleteSubscriptionPublication(id: string): Promise<void> {
  await apiRequest<void>(`${basePath}/${id}`, {
    method: "DELETE",
  });
}

export async function rotateSubscriptionPublicationToken(id: string): Promise<SubscriptionPublication> {
  const data = await apiRequest<ApiPublication>(`${basePath}/${id}/actions/rotate-token`, {
    method: "POST",
  });
  return normalizePublication(data);
}

export async function previewSubscriptionPublication(input: {
  regex_filters: string[];
  region_filters: string[];
  healthy_only: boolean;
}): Promise<SubscriptionPublicationPreview> {
  return apiRequest<SubscriptionPublicationPreview>(`${basePath}/preview`, {
    method: "POST",
    body: input,
  });
}
