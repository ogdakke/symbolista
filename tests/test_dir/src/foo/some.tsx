import { component$, PropsOf } from "@builder.io/qwik";
import { Link as BaseLink } from "@builder.io/qwik-city";
import { cn } from "@qwik-ui/utils";
import { inlineTranslate } from "qwik-speak";
import { Link } from "~/components/ui/link";
import Wordmark from "~/media/wordmark.svg?jsx";

export const Footer = component$<PropsOf<"footer">>((props) => {
  const t = inlineTranslate();
  return (
    <footer {...props} class={cn("isolate", props.class)}>
      <div class="mx-auto mb-8 px-4">
        <div class="rounded-5xl border-border bg-background flex min-h-54 border">
          <div class="grid w-full gap-4 p-4 sm:grid-cols-2 sm:p-8">
            <div class="flex h-full w-fit flex-col justify-between">
              <BaseLink
                href="/"
                data-attr="footer-wordmark-link"
                aria-label="Mene visai:n kotisivulle"
                class="w-fit"
              >
                <Wordmark class="fill-foreground h-10" />
              </BaseLink>
              <div class="flex w-fit flex-col">
                <p class="text-muted-foreground my-4">
                  {t(
                    "visai tarjoaa sinulle avaimet tehokkaaseen opiskeluun, oli aihe mikä tahansa",
                  )}
                </p>
                <Link data-attr="footer-cta" class="self-end" href="/signup" look="primary">
                  {t("Aloita nyt")}
                </Link>
              </div>
            </div>
            <div>
              <div class="flex size-full flex-col justify-between">
                <div class="flex flex-col gap-8 sm:flex-row">
                  <div>
                    <p class="p-2 text-lg font-medium">{t("app.product@@Tuote")}</p>
                    <ul>
                      <li>
                        <Link href="/blog" size="sm">
                          {t("app.blog.title@@Blogi")}
                        </Link>
                      </li>
                      <li>
                        <Link href="/changelog" size="sm">
                          {t("app.changelog@@Versiohistoria")}
                        </Link>
                      </li>
                    </ul>
                  </div>
                  <div>
                    <p class="p-2 text-lg font-medium">{t("app.more_info@@Lisätietoja")}</p>
                    <ul>
                      <li>
                        <Link href="/contact" size="sm">
                          {t("app.contact.title@@Yhteystiedot")}
                        </Link>
                      </li>
                      <li>
                        <Link href="/#features" size="sm">
                          {t("app.features@@Ominaisuudet")}
                        </Link>
                      </li>
                      <li>
                        <Link href="/#pricing" size="sm">
                          {t("app.pricing@@Hinnasto")}
                        </Link>
                      </li>
                      <li>
                        <Link href="/privacy" size="sm">
                          {t("app.privacy@@Yksityisyys")}
                        </Link>
                      </li>
                      <li>
                        <Link href="/tos" size="sm">
                          {t("app.tos.title@@Käyttöehdot")}
                        </Link>
                      </li>
                    </ul>
                  </div>
                </div>
                <div class="flex justify-end">
                  <p>©{t("app.copyright@@visai.fi")} 2025</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
});
