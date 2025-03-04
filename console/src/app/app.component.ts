import { BreakpointObserver } from '@angular/cdk/layout';
import { OverlayContainer } from '@angular/cdk/overlay';
import { DOCUMENT, ViewportScroller } from '@angular/common';
import { Component, HostBinding, HostListener, Inject, OnDestroy, ViewChild } from '@angular/core';
import { MatIconRegistry } from '@angular/material/icon';
import { MatDrawer } from '@angular/material/sidenav';
import { DomSanitizer } from '@angular/platform-browser';
import { ActivatedRoute, Router, RouterOutlet } from '@angular/router';
import { LangChangeEvent, TranslateService } from '@ngx-translate/core';
import { Observable, of, Subject } from 'rxjs';
import { map, take, takeUntil } from 'rxjs/operators';

import { accountCard, adminLineAnimation, navAnimations, routeAnimations, toolbarAnimation } from './animations';
import { Org } from './proto/generated/zitadel/org_pb';
import { LabelPolicy, PrivacyPolicy } from './proto/generated/zitadel/policy_pb';
import { AuthenticationService } from './services/authentication.service';
import { GrpcAuthService } from './services/grpc-auth.service';
import { KeyboardShortcutsService } from './services/keyboard-shortcuts/keyboard-shortcuts.service';
import { ManagementService } from './services/mgmt.service';
import { NavigationService } from './services/navigation.service';
import { OverlayWorkflowService } from './services/overlay/overlay-workflow.service';
import { StorageService } from './services/storage.service';
import { ThemeService } from './services/theme.service';
import { UpdateService } from './services/update.service';

@Component({
  selector: 'cnsl-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  animations: [toolbarAnimation, ...navAnimations, accountCard, routeAnimations, adminLineAnimation],
})
export class AppComponent implements OnDestroy {
  @ViewChild('drawer') public drawer!: MatDrawer;
  public isHandset$: Observable<boolean> = this.breakpointObserver.observe('(max-width: 599px)').pipe(
    map((result) => {
      return result.matches;
    }),
  );
  @HostBinding('class') public componentCssClass: string = 'dark-theme';

  public yoffset: number = 0;
  @HostListener('window:scroll', ['$event']) onScroll(event: Event): void {
    this.yoffset = this.viewPortScroller.getScrollPosition()[1];
  }
  public org!: Org.AsObject;
  public orgs$: Observable<Org.AsObject[]> = of([]);
  public showAccount: boolean = false;
  public isDarkTheme: Observable<boolean> = of(true);

  public showProjectSection: boolean = false;

  private destroy$: Subject<void> = new Subject();
  public labelpolicy!: LabelPolicy.AsObject;

  public language: string = 'en';
  public privacyPolicy!: PrivacyPolicy.AsObject;
  constructor(
    @Inject('windowObject') public window: Window,
    public viewPortScroller: ViewportScroller,
    public translate: TranslateService,
    public authenticationService: AuthenticationService,
    public authService: GrpcAuthService,
    private breakpointObserver: BreakpointObserver,
    public overlayContainer: OverlayContainer,
    private themeService: ThemeService,
    public mgmtService: ManagementService,
    public matIconRegistry: MatIconRegistry,
    public domSanitizer: DomSanitizer,
    private router: Router,
    update: UpdateService,
    keyboardShortcuts: KeyboardShortcutsService,
    private activatedRoute: ActivatedRoute,
    private workflowService: OverlayWorkflowService,
    private storageService: StorageService,
    private navigationService: NavigationService,
    @Inject(DOCUMENT) private document: Document,
  ) {
    console.log(
      '%cWait!',
      'text-shadow: -1px 0 black, 0 1px black, 1px 0 black, 0 -1px black; color: #5469D4; font-size: 50px',
    );
    console.log(
      '%cInserting something here could give attackers access to your zitadel account.',
      'color: red; font-size: 18px',
    );
    console.log(
      "%cIf you don't know exactly what you're doing, close the window and stay on the safe side",
      'font-size: 16px',
    );
    console.log('%cIf you know exactly what you are doing, you should work for us', 'font-size: 16px');
    this.setLanguage();

    this.matIconRegistry.addSvgIcon(
      'mdi_account_check_outline',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/account-check-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_account_cancel',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/account-cancel-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_light_on',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/lightbulb-on-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_content_copy',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/content-copy.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_light_off',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/lightbulb-off-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'usb',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/usb-flash-drive-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon('mdi_radar', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/radar.svg'));

    this.matIconRegistry.addSvgIcon(
      'mdi_lock_question',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/lock-question.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_textbox_password',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/textbox-password.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_lock_reset',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/lock-reset.svg'),
    );

    this.matIconRegistry.addSvgIcon('mdi_broom', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/broom.svg'));

    this.matIconRegistry.addSvgIcon(
      'mdi_pin_outline',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/pin-outline.svg'),
    );

    this.matIconRegistry.addSvgIcon('mdi_pin', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/pin.svg'));

    this.matIconRegistry.addSvgIcon(
      'mdi_format-letter-case-lower',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/format-letter-case-lower.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_format-letter-case-upper',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/format-letter-case-upper.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_counter',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/counter.svg'),
    );

    this.matIconRegistry.addSvgIcon('mdi_openid', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/openid.svg'));

    this.matIconRegistry.addSvgIcon('mdi_jwt', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/jwt.svg'));

    this.matIconRegistry.addSvgIcon('mdi_symbol', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/symbol.svg'));

    this.matIconRegistry.addSvgIcon(
      'mdi_shield_alert',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/shield-alert.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_numeric',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/numeric.svg'),
    );

    this.matIconRegistry.addSvgIcon('mdi_api', this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/api.svg'));

    this.matIconRegistry.addSvgIcon(
      'mdi_arrow_right_bottom',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/arrow-right-bottom.svg'),
    );

    this.matIconRegistry.addSvgIcon(
      'mdi_arrow_decision',
      this.domSanitizer.bypassSecurityTrustResourceUrl('assets/mdi/arrow-decision-outline.svg'),
    );

    this.activatedRoute.queryParams.pipe(takeUntil(this.destroy$)).subscribe((route) => {
      const { org } = route;
      if (org) {
        this.authService
          .getActiveOrg(org)
          .then((queriedOrg) => {
            this.org = queriedOrg;
          })
          .catch((error) => {
            this.router.navigate(['/users/me']);
          });
      }
    });

    this.getProjectCount();

    this.authService.activeOrgChanged.pipe(takeUntil(this.destroy$)).subscribe((org) => {
      this.org = org;
      this.getProjectCount();
    });

    this.authenticationService.authenticationChanged.pipe(takeUntil(this.destroy$)).subscribe((authenticated) => {
      if (authenticated) {
        this.authService
          .getActiveOrg()
          .then((org) => {
            this.org = org;

            this.loadPrivateLabelling();

            // TODO add when console storage is implemented
            // this.startIntroWorkflow();
          })
          .catch((error) => {
            this.router.navigate(['/users/me']);
          });
      }
    });

    this.isDarkTheme = this.themeService.isDarkTheme;
    this.isDarkTheme.subscribe((dark) => this.onSetTheme(dark ? 'dark-theme' : 'light-theme'));

    this.translate.onLangChange.subscribe((language: LangChangeEvent) => {
      this.document.documentElement.lang = language.lang;
      this.language = language.lang;
    });
  }

  // TODO implement Console storage

  //   private startIntroWorkflow(): void {
  //     setTimeout(() => {
  //       const cb = () => {
  //         this.storageService.setItem('intro-dismissed', true, StorageLocation.local);
  //       };
  //       const dismissed = this.storageService.getItem('intro-dismissed', StorageLocation.local);
  //       if (!dismissed) {
  //         this.workflowService.startWorkflow(IntroWorkflowOverlays, cb);
  //       }
  //     }, 1000);
  //   }

  public ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  public loadPrivateLabelling(): void {
    const setDefaultColors = () => {
      const darkPrimary = '#bbbafa';
      const lightPrimary = '#5469d4';

      const darkWarn = '#ff3b5b';
      const lightWarn = '#cd3d56';

      const darkBackground = '#111827';
      const lightBackground = '#fafafa';

      const darkText = '#ffffff';
      const lightText = '#000000';

      this.themeService.savePrimaryColor(darkPrimary, true);
      this.themeService.savePrimaryColor(lightPrimary, false);

      this.themeService.saveWarnColor(darkWarn, true);
      this.themeService.saveWarnColor(lightWarn, false);

      this.themeService.saveBackgroundColor(darkBackground, true);
      this.themeService.saveBackgroundColor(lightBackground, false);

      this.themeService.saveTextColor(darkText, true);
      this.themeService.saveTextColor(lightText, false);
    };

    setDefaultColors();

    this.mgmtService.getLabelPolicy().then((labelpolicy) => {
      if (labelpolicy.policy) {
        this.labelpolicy = labelpolicy.policy;

        const isDark = (color: string) => this.themeService.isDark(color);
        const isLight = (color: string) => this.themeService.isLight(color);

        const darkPrimary = this.labelpolicy?.primaryColorDark || '#bbbafa';
        const lightPrimary = this.labelpolicy?.primaryColor || '#5469d4';

        const darkWarn = this.labelpolicy?.warnColorDark || '#ff3b5b';
        const lightWarn = this.labelpolicy?.warnColor || '#cd3d56';

        let darkBackground = this.labelpolicy?.backgroundColorDark;
        let lightBackground = this.labelpolicy?.backgroundColor;

        let darkText = this.labelpolicy.fontColorDark;
        let lightText = this.labelpolicy.fontColor;

        this.themeService.savePrimaryColor(darkPrimary, true);
        this.themeService.savePrimaryColor(lightPrimary, false);

        this.themeService.saveWarnColor(darkWarn, true);
        this.themeService.saveWarnColor(lightWarn, false);

        if (darkBackground && !isDark(darkBackground)) {
          console.info(
            `Background (${darkBackground}) is not dark enough for a dark theme. Falling back to zitadel background`,
          );
          darkBackground = '#111827';
        }
        this.themeService.saveBackgroundColor(darkBackground || '#111827', true);

        if (lightBackground && !isLight(lightBackground)) {
          console.info(
            `Background (${lightBackground}) is not light enough for a light theme. Falling back to zitadel background`,
          );
          lightBackground = '#fafafa';
        }
        this.themeService.saveBackgroundColor(lightBackground || '#fafafa', false);

        if (darkText && !isLight(darkText)) {
          console.info(
            `Text color (${darkText}) is not light enough for a dark theme. Falling back to zitadel's text color`,
          );
          darkText = '#ffffff';
        }
        this.themeService.saveTextColor(darkText || '#ffffff', true);

        if (lightText && !isDark(lightText)) {
          console.info(
            `Text color (${lightText}) is not dark enough for a light theme. Falling back to zitadel's text color`,
          );
          lightText = '#000000';
        }
        this.themeService.saveTextColor(lightText || '#000000', false);
      }
    });
  }

  public prepareRoute(outlet: RouterOutlet): boolean {
    return outlet && outlet.activatedRouteData && outlet.activatedRouteData.animation;
  }

  public onSetTheme(theme: string): void {
    localStorage.setItem('theme', theme);
    this.overlayContainer.getContainerElement().classList.remove(theme === 'dark-theme' ? 'light-theme' : 'dark-theme');
    this.overlayContainer.getContainerElement().classList.add(theme);
    this.componentCssClass = theme;
  }

  public changedOrg(org: Org.AsObject): void {
    this.loadPrivateLabelling();
    this.authService.zitadelPermissionsChanged.pipe(take(1)).subscribe(() => {
      this.router.navigate(['/org'], { fragment: org.id });
    });
  }

  private setLanguage(): void {
    this.translate.addLangs(['en', 'de']);
    this.translate.setDefaultLang('en');

    this.authService.user.subscribe((userprofile) => {
      if (userprofile) {
        // this.user = userprofile;
        const cropped = navigator.language.split('-')[0] ?? 'en';
        const fallbackLang = cropped.match(/en|de|it/) ? cropped : 'en';

        const lang = userprofile?.human?.profile?.preferredLanguage.match(/en|de|it/)
          ? userprofile.human.profile?.preferredLanguage
          : fallbackLang;
        this.translate.use(lang);
        this.language = lang;
        this.document.documentElement.lang = lang;
      }
    });
  }

  private getProjectCount(): void {
    this.authService.isAllowed(['project.read']).subscribe((allowed) => {
      if (allowed) {
        this.mgmtService.listProjects(0, 0);
        this.mgmtService.listGrantedProjects(0, 0);
      }
    });
  }
}
