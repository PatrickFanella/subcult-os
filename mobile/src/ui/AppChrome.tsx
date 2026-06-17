import { Link, usePathname } from 'expo-router';
import { Briefcase, Compass, Ticket } from 'lucide-react-native';
import type { PropsWithChildren } from 'react';
import { SafeAreaView, StyleSheet, Text, View } from 'react-native';

const navItems = [
  { href: '/' as const, icon: Compass, label: 'Discover' },
  { href: '/tickets' as const, icon: Ticket, label: 'Tickets' },
  { href: '/staff' as const, icon: Briefcase, label: 'Staff' },
];

export function AppChrome({ children }: PropsWithChildren) {
  const pathname = usePathname();
  const isScanner = pathname.includes('/scanner');

  return (
    <SafeAreaView style={styles.shell}>
      <View style={styles.phone}>
        <View style={styles.content}>{children}</View>
        {!isScanner ? (
          <View style={styles.navWrap}>
            <View style={styles.nav}>
              {navItems.map((item) => (
                <BottomNavItem key={item.href} pathname={pathname} {...item} />
              ))}
            </View>
          </View>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

function BottomNavItem({ href, icon: Icon, label, pathname }: (typeof navItems)[number] & { pathname: string }) {
  const active = href === '/' ? pathname === '/' : pathname.startsWith(href);

  return (
    <Link href={href} style={styles.navItem}>
      <Icon size={24} color={active ? '#171717' : '#a3a3a3'} strokeWidth={2} />
      <Text style={[styles.navLabel, active && styles.navLabelActive]}>{label}</Text>
      <View style={[styles.navDot, active && styles.navDotActive]} />
    </Link>
  );
}

const styles = StyleSheet.create({
  shell: { flex: 1, backgroundColor: '#f5f5f5' },
  phone: { flex: 1, backgroundColor: '#ffffff', overflow: 'hidden' },
  content: { flex: 1, backgroundColor: '#000000' },
  navWrap: {
    backgroundColor: 'rgba(255,255,255,0.9)',
    borderTopWidth: 1,
    borderTopColor: '#f5f5f5',
    paddingBottom: 18,
    zIndex: 40,
  },
  nav: { height: 78, paddingHorizontal: 16, paddingBottom: 8, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-around' },
  navItem: { width: 80, alignItems: 'center', gap: 4 },
  navLabel: { fontSize: 10, fontWeight: '500', letterSpacing: 0.2, color: '#a3a3a3' },
  navLabelActive: { color: '#171717' },
  navDot: { width: 4, height: 4, borderRadius: 2, backgroundColor: 'transparent', marginTop: 2 },
  navDotActive: { backgroundColor: '#171717' },
});
