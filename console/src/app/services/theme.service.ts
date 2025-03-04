import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

declare const tinycolor: any;

export interface Color {
  name: string;
  hex: string;
  rgb: string;
  contrastColor: string;
}

@Injectable()
export class ThemeService {
  private _darkTheme: BehaviorSubject<boolean> = new BehaviorSubject<boolean>(true);
  public isDarkTheme: Observable<boolean> = this._darkTheme.asObservable();

  private primaryColorPalette: Color[] = [];
  private warnColorPalette: Color[] = [];
  private backgroundColorPalette: Color[] = [];

  constructor() {
    const theme = localStorage.getItem('theme');
    if (theme) {
      if (theme === 'light-theme') {
        this.setDarkTheme(false);
      } else {
        this.setDarkTheme(true);
      }
    }
  }

  setDarkTheme(isDarkTheme: boolean): void {
    this._darkTheme.next(isDarkTheme);
  }

  public updateTheme(colors: Color[], type: string, theme: string): void {
    colors.forEach((color) => {
      document.documentElement.style.setProperty(`--theme-${theme}-${type}-${color.name}`, color.hex);
      document.documentElement.style.setProperty(`--theme-${theme}-${type}-contrast-${color.name}`, color.contrastColor);
    });
  }

  public savePrimaryColor(color: string, isDark: boolean): void {
    this.primaryColorPalette = this.computeColors(color);
    this.updateTheme(this.primaryColorPalette, 'primary', isDark ? 'dark' : 'light');
  }

  public saveWarnColor(color: string, isDark: boolean): void {
    this.warnColorPalette = this.computeColors(color);
    this.updateTheme(this.warnColorPalette, 'warn', isDark ? 'dark' : 'light');
  }

  public saveBackgroundColor(color: string, isDark: boolean): void {
    this.backgroundColorPalette = this.computeColors(color);
    this.updateTheme(this.backgroundColorPalette, 'background', isDark ? 'dark' : 'light');
  }

  public saveTextColor(colorHex: string, isDark: boolean): void {
    const theme = isDark ? 'dark' : 'light';
    document.documentElement.style.setProperty(`--theme-${theme}-${'text'}`, colorHex);
    const secondaryTextHex = tinycolor(colorHex).setAlpha(0.78).toHex8String();
    document.documentElement.style.setProperty(`--theme-${theme}-${'secondary-text'}`, secondaryTextHex);
  }

  private computeColors(hex: string): Color[] {
    return [
      this.getColorObject(tinycolor(hex).lighten(52), '50'),
      this.getColorObject(tinycolor(hex).lighten(37), '100'),
      this.getColorObject(tinycolor(hex).lighten(26), '200'),
      this.getColorObject(tinycolor(hex).lighten(12), '300'),
      this.getColorObject(tinycolor(hex).lighten(6), '400'),
      this.getColorObject(tinycolor(hex), '500'),
      this.getColorObject(tinycolor(hex).darken(6), '600'),
      this.getColorObject(tinycolor(hex).darken(12), '700'),
      this.getColorObject(tinycolor(hex).darken(18), '800'),
      this.getColorObject(tinycolor(hex).darken(24), '900'),
      this.getColorObject(tinycolor(hex).lighten(50).saturate(30), 'A100'),
      this.getColorObject(tinycolor(hex).lighten(30).saturate(30), 'A200'),
      this.getColorObject(tinycolor(hex).lighten(10).saturate(15), 'A400'),
      this.getColorObject(tinycolor(hex).lighten(5).saturate(5), 'A700'),
    ];
  }

  private getColorObject(value: any, name: string): Color {
    const c = tinycolor(value);
    return {
      name: name,
      hex: c.toHexString(),
      rgb: c.toRgbString(),
      contrastColor: this.getContrast(c.toHexString()),
    };
  }

  public isLight(hex: string): boolean {
    const color = tinycolor(hex);
    return color.isLight();
  }

  public isDark(hex: string): boolean {
    const color = tinycolor(hex);
    return color.isDark();
  }

  public getContrast(color: string): string {
    const onBlack = tinycolor.readability('#000', color);
    const onWhite = tinycolor.readability('#fff', color);
    if (onBlack > onWhite) {
      return 'hsla(0, 0%, 0%, 0.87)';
    } else {
      return '#ffffff';
    }
  }
}
