import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import {
  AlertTriangle,
  Check,
  Copy,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Sparkles,
  Trash2,
  X,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { DataTable } from "../../components/ui/DataTable";
import { Input } from "../../components/ui/Input";
import { OffsetPagination } from "../../components/ui/OffsetPagination";
import { Select } from "../../components/ui/Select";
import { Switch } from "../../components/ui/Switch";
import { ToastContainer } from "../../components/ui/Toast";
import { useToast } from "../../hooks/useToast";
import { useI18n } from "../../i18n";
import { formatApiErrorMessage } from "../../lib/error-message";
import { formatRelativeTime } from "../../lib/time";
import {
  createSubscriptionPublication,
  deleteSubscriptionPublication,
  listSubscriptionPublications,
  rotateSubscriptionPublicationToken,
  updateSubscriptionPublication,
} from "./api";
import {
  toCreateInput,
  toUpdateInput,
  type PublicationFormValues,
} from "./formModel";
import { PublicationFormModal } from "./PublicationFormModal";
import type { SubscriptionPublication } from "./types";
import { resolvePublicURL } from "./urls";

type EnabledFilter = "all" | "enabled" | "disabled";

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const;
const EMPTY_PUBLICATIONS: SubscriptionPublication[] = [];

function parseEnabledFilter(value: EnabledFilter): boolean | undefined {
  if (value === "enabled") {
    return true;
  }
  if (value === "disabled") {
    return false;
  }
  return undefined;
}

function CopyIconButton({ value }: { value: string }) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value);
      } else {
        const area = document.createElement("textarea");
        area.value = value;
        area.style.position = "fixed";
        area.style.opacity = "0";
        document.body.appendChild(area);
        area.select();
        document.execCommand("copy");
        document.body.removeChild(area);
      }
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label={t("复制地址")}
      title={t("复制地址")}
      onClick={(event) => {
        event.stopPropagation();
        void handleCopy();
      }}
    >
      {copied ? <Check size={14} /> : <Copy size={14} />}
    </Button>
  );
}

function ConfirmModal({
  title,
  description,
  confirmLabel,
  danger,
  pending,
  onCancel,
  onConfirm,
}: {
  title: string;
  description: string;
  confirmLabel: string;
  danger?: boolean;
  pending: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <Card className="modal-card" style={{ maxWidth: 480 }}>
        <div className="modal-header">
          <h3>{title}</h3>
          <Button type="button" variant="ghost" size="sm" aria-label={t("关闭")} onClick={onCancel}>
            <X size={16} />
          </Button>
        </div>
        <p className="muted">{description}</p>
        <div className="detail-actions">
          <Button type="button" variant={danger ? "danger" : "primary"} disabled={pending} onClick={onConfirm}>
            {pending ? t("处理中...") : confirmLabel}
          </Button>
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("取消")}
          </Button>
        </div>
      </Card>
    </div>
  );
}

export function PublicationPage() {
  const { t } = useI18n();
  const { toasts, showToast, dismissToast } = useToast();
  const queryClient = useQueryClient();

  const [enabledFilter, setEnabledFilter] = useState<EnabledFilter>("all");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState<number>(20);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<SubscriptionPublication | null>(null);
  const [deleting, setDeleting] = useState<SubscriptionPublication | null>(null);
  const [rotating, setRotating] = useState<SubscriptionPublication | null>(null);

  const enabledValue = parseEnabledFilter(enabledFilter);
  const publicationsQuery = useQuery({
    queryKey: ["subscription-publications", enabledFilter, page, pageSize, search],
    queryFn: () =>
      listSubscriptionPublications({
        enabled: enabledValue,
        limit: pageSize,
        offset: page * pageSize,
        keyword: search,
      }),
    refetchInterval: 60_000,
    placeholderData: (prev) => prev,
  });

  const publications = publicationsQuery.data?.items ?? EMPTY_PUBLICATIONS;
  const totalPublications = publicationsQuery.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalPublications / pageSize));
  const currentPage = Math.min(page, totalPages - 1);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["subscription-publications"] });

  const createMutation = useMutation({
    mutationFn: createSubscriptionPublication,
    onSuccess: async (created) => {
      await invalidate();
      setCreateOpen(false);
      showToast("success", t("发布 {{name}} 创建成功", { name: created.name }));
    },
    onError: (error) => showToast("error", formatApiErrorMessage(error, t)),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: PublicationFormValues }) =>
      updateSubscriptionPublication(id, toUpdateInput(values)),
    onSuccess: async (updated) => {
      await invalidate();
      setEditing(null);
      showToast("success", t("发布 {{name}} 已更新", { name: updated.name }));
    },
    onError: (error) => showToast("error", formatApiErrorMessage(error, t)),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      updateSubscriptionPublication(id, { enabled }),
    onSuccess: async () => {
      await invalidate();
    },
    onError: (error) => showToast("error", formatApiErrorMessage(error, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSubscriptionPublication,
    onSuccess: async () => {
      await invalidate();
      setDeleting(null);
      showToast("success", t("发布已删除"));
    },
    onError: (error) => showToast("error", formatApiErrorMessage(error, t)),
  });

  const rotateMutation = useMutation({
    mutationFn: rotateSubscriptionPublicationToken,
    onSuccess: async (updated) => {
      await invalidate();
      setRotating(null);
      setEditing(updated);
      showToast("success", t("令牌已重新生成，旧地址立即失效"));
    },
    onError: (error) => showToast("error", formatApiErrorMessage(error, t)),
  });

  const columns = useMemo(() => {
    const helper = createColumnHelper<SubscriptionPublication>();
    return [
      helper.accessor("name", {
        header: t("名称"),
        cell: (info) => {
          const publication = info.row.original;
          return (
            <div className="publication-name-cell">
              <strong>{publication.name}</strong>
              <Badge variant={publication.enabled ? "success" : "muted"}>
                {publication.enabled ? t("已启用") : t("已停用")}
              </Badge>
            </div>
          );
        },
      }),
      helper.display({
        id: "filters",
        header: t("过滤规则"),
        cell: (info) => {
          const publication = info.row.original;
          return (
            <span className="muted">
              {t("{{regex}} 条正则 / {{region}} 条地区", {
                regex: publication.regex_filters.length,
                region: publication.region_filters.length,
              })}
            </span>
          );
        },
      }),
      helper.display({
        id: "nodes",
        header: t("节点数"),
        cell: (info) => {
          const publication = info.row.original;
          return (
            <span title={t("匹配 {{matching}}，跳过 {{skipped}}，健康 {{healthy}}", {
              matching: publication.matching_node_count,
              skipped: publication.skipped_node_count,
              healthy: publication.healthy_node_count,
            })}>
              {publication.node_count}
            </span>
          );
        },
      }),
      helper.display({
        id: "status",
        header: t("状态"),
        cell: (info) => {
          const publication = info.row.original;
          return (
            <span onClick={(event) => event.stopPropagation()}>
              <Switch
                checked={publication.enabled}
                onChange={(event) =>
                  toggleMutation.mutate({ id: publication.id, enabled: event.target.checked })
                }
              />
            </span>
          );
        },
      }),
      helper.display({
        id: "address",
        header: t("订阅地址"),
        cell: (info) => {
          const publication = info.row.original;
          const url = resolvePublicURL(publication.subscription_path);
          return (
            <div className="publication-url-cell">
              <code className="publication-url-text" title={url}>
                {url}
              </code>
              <CopyIconButton value={url} />
            </div>
          );
        },
      }),
      helper.accessor("updated_at", {
        header: t("更新时间"),
        cell: (info) => <span className="muted">{formatRelativeTime(info.getValue())}</span>,
      }),
      helper.display({
        id: "actions",
        header: t("操作"),
        cell: (info) => {
          const publication = info.row.original;
          return (
            <div className="publication-actions" onClick={(event) => event.stopPropagation()}>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={t("编辑")}
                title={t("编辑")}
                onClick={() => setEditing(publication)}
              >
                <Pencil size={14} />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={t("重新生成令牌")}
                title={t("重新生成令牌")}
                onClick={() => setRotating(publication)}
              >
                <KeyRound size={14} />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={t("删除发布")}
                title={t("删除发布")}
                onClick={() => setDeleting(publication)}
              >
                <Trash2 size={14} />
              </Button>
            </div>
          );
        },
      }),
    ];
  }, [t, toggleMutation]);

  const changePageSize = (next: number) => {
    setPageSize(next);
    setPage(0);
  };

  const handleCreate = async (values: PublicationFormValues) => {
    try {
      await createMutation.mutateAsync(toCreateInput(values));
    } catch {
      // 错误提示由 mutation onError 处理，保持弹窗打开。
    }
  };

  const handleUpdate = async (values: PublicationFormValues) => {
    if (!editing) {
      return;
    }
    try {
      await updateMutation.mutateAsync({ id: editing.id, values });
    } catch {
      // 错误提示由 mutation onError 处理，保持弹窗打开。
    }
  };

  return (
    <section className="platform-page">
      <header className="module-header">
        <div>
          <h2>{t("订阅发布")}</h2>
          <p className="module-description">
            {t("按节点标签与地区规则生成 Clash / sing-box 订阅地址，供现代客户端直接订阅。")}
          </p>
        </div>
      </header>

      <ToastContainer toasts={toasts} onDismiss={dismissToast} />

      <Card className="platform-list-card platform-directory-card">
        <div className="list-card-header">
          <div>
            <h3>{t("发布列表")}</h3>
            <p>{t("共 {{count}} 个发布", { count: totalPublications })}</p>
          </div>
          <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            <Select
              value={enabledFilter}
              onChange={(event) => {
                setEnabledFilter(event.target.value as EnabledFilter);
                setPage(0);
              }}
              style={{ width: 120 }}
            >
              <option value="all">{t("全部状态")}</option>
              <option value="enabled">{t("已启用")}</option>
              <option value="disabled">{t("已停用")}</option>
            </Select>
            <label className="search-box" htmlFor="publication-search" style={{ maxWidth: 200, margin: 0, gap: 6 }}>
              <Search size={16} />
              <Input
                id="publication-search"
                placeholder={t("搜索发布")}
                value={search}
                onChange={(event) => {
                  setSearch(event.target.value);
                  setPage(0);
                }}
                style={{ padding: "6px 10px", borderRadius: 8 }}
              />
            </label>
            <Button type="button" variant="secondary" size="sm" onClick={() => setCreateOpen(true)}>
              <Plus size={16} />
              {t("新建")}
            </Button>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => publicationsQuery.refetch()}
              disabled={publicationsQuery.isFetching}
            >
              <RefreshCw size={16} className={publicationsQuery.isFetching ? "spin" : undefined} />
              {t("刷新")}
            </Button>
          </div>
        </div>
      </Card>

      <Card className="platform-cards-container subscriptions-table-card">
        {publicationsQuery.isLoading ? <p className="muted">{t("正在加载发布数据...")}</p> : null}

        {publicationsQuery.isError ? (
          <div className="callout callout-error">
            <AlertTriangle size={14} />
            <span>{formatApiErrorMessage(publicationsQuery.error, t)}</span>
          </div>
        ) : null}

        {!publicationsQuery.isLoading && !publications.length ? (
          <div className="empty-box">
            <Sparkles size={16} />
            <p>{t("没有匹配的发布")}</p>
          </div>
        ) : null}

        {publications.length ? (
          <DataTable
            data={publications}
            columns={columns}
            onRowClick={(publication) => setEditing(publication)}
            getRowId={(publication) => publication.id}
            className="data-table-subs"
          />
        ) : null}

        <OffsetPagination
          page={currentPage}
          totalPages={totalPages}
          totalItems={totalPublications}
          pageSize={pageSize}
          pageSizeOptions={PAGE_SIZE_OPTIONS}
          onPageChange={setPage}
          onPageSizeChange={changePageSize}
        />
      </Card>

      {createOpen ? (
        <PublicationFormModal
          mode="create"
          submitting={createMutation.isPending}
          onClose={() => setCreateOpen(false)}
          onSubmit={handleCreate}
        />
      ) : null}

      {editing ? (
        <PublicationFormModal
          mode="edit"
          publication={editing}
          submitting={updateMutation.isPending}
          onClose={() => setEditing(null)}
          onSubmit={handleUpdate}
        />
      ) : null}

      {deleting ? (
        <ConfirmModal
          title={t("删除发布")}
          description={t("删除发布 {{name}}？该地址将立即失效，节点池不会受到影响。", { name: deleting.name })}
          confirmLabel={t("确认删除")}
          danger
          pending={deleteMutation.isPending}
          onCancel={() => setDeleting(null)}
          onConfirm={() => deleteMutation.mutate(deleting.id)}
        />
      ) : null}

      {rotating ? (
        <ConfirmModal
          title={t("重新生成令牌")}
          description={t("重新生成发布 {{name}} 的令牌？旧订阅地址会立即失效。", { name: rotating.name })}
          confirmLabel={t("确认重新生成")}
          pending={rotateMutation.isPending}
          onCancel={() => setRotating(null)}
          onConfirm={() => rotateMutation.mutate(rotating.id)}
        />
      ) : null}
    </section>
  );
}
