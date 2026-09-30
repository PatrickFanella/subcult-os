import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link, usePathname } from 'expo-router';
import { Briefcase, Compass, Ticket, UserCircle } from 'lucide-react-native';
import type { PropsWithChildren } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const navItems = [
  { href: '/' as const, icon: Compass, label: 'Discover' },
  { href: '/tickets' as const, icon: Ticket, label: 'Tickets' },
  { href: '/staff' as const, icon: Briefcase, label: 'Staff' },
  { href: '/profile' as const, icon: UserCircle, label: 'Profile' },
];

export function AppChrome({ children }: PropsWithChildren) {

  const styles = useThemedStyles(createStyles);

  const pathname = usePathname();
  const insets = useSafeAreaInsets();
  const isScanner = pathname.includes('/scanner');

  return (
    <View style={styles.shell}>
      <View style={styles.phone}>
        <View style={styles.content}>{children}</View>
        {!isScanner ? (
          <View style={[styles.navWrap, { paddingBottom: insets.bottom }]}>
            <View style={styles.nav}>
              {navItems.map((item) => (
                <BottomNavItem key={item.href} pathname={pathname} {...item} />
              ))}
            </View>
          </View>
        ) : null}
      </View>
    </View>
  );
}

function BottomNavItem({ href, icon: Icon, label, pathname }: (typeof navItems)[number] & { pathname: string }) {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const active = href === '/' ? pathname === '/' : pathname.startsWith(href) || (href === '/profile' && pathname.startsWith('/settings'));

  return (
    <Link href={href} asChild>
      <Pressable style={styles.navItem} accessibilityRole="tab" accessibilityLabel={label} accessibilityState={{ selected: active }}>
        <Icon size={23} color={active ? tokens.color.text.primary : tokens.color.text.muted} strokeWidth={2} />
        <Text style={[styles.navLabel, active && styles.navLabelActive]}>{label}</Text>
        <View style={[styles.navDot, active && styles.navDotActive]} />
      </Pressable>
    </Link>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  shell: { flex: 1, backgroundColor: tokens.color.surface.inset },
  phone: { flex: 1, backgroundColor: tokens.color.surface.panel, overflow: 'hidden' },
  content: { flex: 1, backgroundColor: tokens.color.surface.immersive },
  navWrap: {
    backgroundColor: tokens.color.surface.panel,
    borderTopWidth: 1,
    borderTopColor: tokens.color.surface.inset,
    zIndex: 40,
  },
  nav: { height: 68, paddingHorizontal: 8, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-around' },
  navItem: { width: 72, height: 58, alignItems: 'center', justifyContent: 'center', gap: 3 },
  navLabel: { fontSize: 10, fontWeight: '500', letterSpacing: 0.2, color: tokens.color.text.muted },
  navLabelActive: { color: tokens.color.text.primary },
  navDot: { width: 4, height: 4, borderRadius: 2, backgroundColor: 'transparent', marginTop: 2 },
  navDotActive: { backgroundColor: tokens.color.action.primary },
});
