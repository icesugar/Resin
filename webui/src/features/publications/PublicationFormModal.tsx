import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Copy, Info, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Input } from "../../components/ui/Input";
import { Switch } from "../../components/ui/Switch";
import { Textarea } from "../../components/ui/Textarea";
import { useI18n } from "../../i18n";
import {
  defaultPublicationFormValues,
  publicationFormSchema,
  publicationNameRuleHint,
  publicationToFormValues,
  type PublicationFormValues,
} from "./formModel";
import type { SubscriptionPublication } from "./types";
import { resolvePublicURL } from "./urls";

type PublicationFormModalProps = {
  mode: "create" | "edit";
  publication?: SubscriptionPublication | null;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (values: PublicationFormValues) => Promise<void>;
};

type CopyValueProps = {
  label: string;
  value: string;
  hint?: string;
};

function CopyValue({ label, value, hint }: CopyValueProps) {
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
    <div className="platform-access-field">
      <div className="platform-access-field-head">
        <span className="platform-access-field-label">{label}</span>
        {hint ? <span className="platform-access-field-hint">{hint}</span> : null}
      </div>
      <div className="platform-access-field-body">
        <code className="platform-access-value" title={value}>
          {value}
        </code>
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => void handleCopy()}
          className="platform-access-copy-btn"
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          {copied ? t("已复制") : t("复制地址")}
        </Button>
      </div>
    </div>
  );
}

export function PublicationFormModal({
  mode,
  publication,
  submitting,
  onClose,
  onSubmit,
}: PublicationFormModalProps) {
  const { t } = useI18n();
  const form = useForm<PublicationFormValues>({
    resolver: zodResolver(publicationFormSchema),
    defaultValues: defaultPublicationFormValues,
  });

  useEffect(() => {
    form.reset(publication ? publicationToFormValues(publication) : defaultPublicationFormValues);
  }, [publication, form]);

  const submit = form.handleSubmit(async (values) => {
    await onSubmit(values);
  });

  const title = mode === "create" ? t("新建发布") : t("编辑发布 {{name}}", { name: publication?.name ?? "" });

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <Card className="modal-card">
        <div className="modal-header">
          <h3>{title}</h3>
          <Button type="button" variant="ghost" size="sm" aria-label={t("关闭")} onClick={onClose}>
            <X size={16} />
          </Button>
        </div>

        <form className="form-grid" onSubmit={submit}>
          <div className="field-group">
            <label className="field-label" htmlFor="publication-name">
              {t("名称")}
            </label>
            <Input id="publication-name" invalid={Boolean(form.formState.errors.name)} {...form.register("name")} />
            {form.formState.errors.name?.message ? (
              <p className="field-error">{t(form.formState.errors.name.message)}</p>
            ) : null}
            <p className="muted" style={{ marginTop: 4, fontSize: 12 }}>
              {t(publicationNameRuleHint)}
            </p>
          </div>

          <div className="field-group">
            <label className="field-label" htmlFor="publication-enabled">
              {t("启用发布")}
            </label>
            <div className="subscription-switch-item">
              <label className="subscription-switch-label" htmlFor="publication-enabled">
                <span>{t("启用后该订阅地址才可访问")}</span>
              </label>
              <Switch id="publication-enabled" {...form.register("enabled")} />
            </div>
          </div>

          <div className="field-group">
            <label className="field-label" htmlFor="publication-healthy">
              {t("仅导出健康节点")}
            </label>
            <div className="subscription-switch-item">
              <label className="subscription-switch-label" htmlFor="publication-healthy">
                <span>{t("仅导出健康节点")}</span>
                <span
                  className="subscription-info-icon"
                  title={t("开启后仅导出出口已就绪且未熔断的节点；客户端仍会自行测速。")}
                  aria-label={t("开启后仅导出出口已就绪且未熔断的节点；客户端仍会自行测速。")}
                  tabIndex={0}
                >
                  <Info size={13} />
                </span>
              </label>
              <Switch id="publication-healthy" {...form.register("healthy_only")} />
            </div>
          </div>

          <div className="field-group">
            <span className="field-label">{t("发布格式")}</span>
            <div className="publication-format-note">
              <div className="publication-format-line">
                <span>{t("Clash / Mihomo")}</span>
                <code>.yaml</code>
              </div>
              <div className="publication-format-line">
                <span>{t("sing-box")}</span>
                <code>.json</code>
              </div>
            </div>
          </div>

          <div className="field-group">
            <label className="field-label" htmlFor="publication-regex">
              {t("节点名正则过滤规则（可选）")}
            </label>
            <Textarea
              id="publication-regex"
              rows={6}
              placeholder={t("每行一条正则表达式，例如：\n\n香港\n日本\n*专线\n!过期\n!失效\n\n表示：选择【香港】或【日本】的【专线】节点，并排除包含【过期】或【失效】的节点。")}
              {...form.register("regex_filters_text")}
            />
            <div className="muted" style={{ marginTop: 4, fontSize: 12 }}>
              <div>{t("普通正则表达式表示满足其一，* 开头表示必须包含，! 开头表示排除。")}</div>
              <div>{t("技巧：^<订阅名>/ 可筛选来自该订阅的节点。")}</div>
            </div>
          </div>

          <div className="field-group">
            <label className="field-label" htmlFor="publication-region">
              {t("地区过滤规则（可选）")}
            </label>
            <Textarea
              id="publication-region"
              rows={6}
              placeholder={t("每行一条，如 hk / us / !hk")}
              {...form.register("region_filters_text")}
            />
            <p className="muted" style={{ marginTop: 4, fontSize: 12 }}>
              {t("支持反选：以 ! 开头可排除地区（如 !hk）。可与正选混用，最终结果为“先正选再排除”。")}
            </p>
          </div>

          {publication ? (
            <div className="field-group field-span-2">
              <CopyValue label={t("Clash 订阅地址")} value={resolvePublicURL(publication.subscription_path)} />
              <CopyValue
                label={t("Clash Provider 地址")}
                value={resolvePublicURL(publication.provider_path)}
                hint={t("仅包含 proxies，适合作为 Provider 使用")}
              />
              <CopyValue label={t("sing-box 地址")} value={resolvePublicURL(publication.singbox_path)} />
              <p className="muted" style={{ marginTop: 4, fontSize: 12 }}>
                {t("Clash / Mihomo / OpenClash / PassWall / Shadowrocket 使用 .yaml 地址；sing-box 使用 .json 地址。")}
              </p>
            </div>
          ) : null}

          <div className="detail-actions">
            <Button type="submit" disabled={submitting}>
              {submitting ? t("保存中...") : mode === "create" ? t("确认创建") : t("保存修改")}
            </Button>
            <Button type="button" variant="secondary" onClick={onClose}>
              {t("取消")}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
