export type PageResponse<T> = {
  items: T[];
  total: number;
  limit: number;
  offset: number;
};

export type SubscriptionPublication = {
  id: string;
  name: string;
  enabled: boolean;
  regex_filters: string[];
  region_filters: string[];
  healthy_only: boolean;
  matching_node_count: number;
  node_count: number;
  skipped_node_count: number;
  healthy_node_count: number;
  subscription_path: string;
  provider_path: string;
  singbox_path: string;
  created_at: string;
  updated_at: string;
};

export type SubscriptionPublicationCreateInput = {
  name: string;
  enabled?: boolean;
  regex_filters?: string[];
  region_filters?: string[];
  healthy_only?: boolean;
};

export type SubscriptionPublicationUpdateInput = {
  name?: string;
  enabled?: boolean;
  regex_filters?: string[];
  region_filters?: string[];
  healthy_only?: boolean;
};

export type SubscriptionPublicationPreview = {
  matching_node_count: number;
  node_count: number;
  skipped_node_count: number;
  healthy_node_count: number;
  sample_names: string[];
};
