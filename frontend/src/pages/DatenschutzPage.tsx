export function DatenschutzPage() {
  return (
    <article className="prose" data-od-id="datenschutz-content">
      <h1 className="page__title">Datenschutzerklärung</h1>

      <h2>Verantwortlicher</h2>
      <p>
        Kfz-Meisterbetrieb Vogel, Inhaber Thomas Vogel, Vogelstraße 12, 10115 Berlin. Telefon:
        030 123 456 78, E-Mail:{' '}
        <a href="mailto:datenschutz@werkstatt-vogel.de">datenschutz@werkstatt-vogel.de</a>.
      </p>

      <h2>Zweck der Verarbeitung</h2>
      <p>
        Wir verarbeiten die im Kundenportal erhobenen Daten ausschließlich, um Ihre Terminanfrage
        zu bearbeiten, Ihren Werkstattauftrag durchzuführen, den Status abzufragen und die
        Rechnung zu erstellen und bereitzustellen. Eine Nutzung zu anderen Zwecken findet nicht
        statt.
      </p>

      <h2>Welche Daten wir verarbeiten</h2>
      <p>Über das Kundenportal erheben wir folgende personenbezogene Daten:</p>
      <ul>
        <li>
          <strong>Name</strong> — zur Zuordnung Ihrer Anfrage und Ihres Auftrags.
        </li>
        <li>
          <strong>E-Mail</strong> — zur Bestätigung und für Rückfragen zu Ihrem Auftrag.
        </li>
        <li>
          <strong>Telefon</strong> — für kurzfristige Absprachen zum Werkstatttermin.
        </li>
        <li>
          <strong>Kennzeichen</strong> — zur Zuordnung Ihres Fahrzeugs und Ihrer Aufträge.
        </li>
      </ul>
      <p>
        Zusätzlich speichern wir die zum Auftrag gehörenden Fahrzeug- und Auftragsdaten (z. B.
        Marke, Modell, Kilometerstand, Wunschtermin, Problembeschreibung, Positionen und
        Rechnungsbeträge).
      </p>

      <h2>Speicherdauer</h2>
      <p>
        Ihre Kontaktdaten und die Auftragsdaten speichern wir für die Dauer der Bearbeitung und
        anschließend so lange, wie es die handels- und steuerrechtlichen Aufbewahrungsfristen (in
        der Regel bis zu zehn Jahre für Rechnungen) vorschreiben. Nach Ablauf der Fristen werden
        die Daten gelöscht.
      </p>

      <h2>Kontaktweg für Fragen und zur Wahrnehmung Ihrer Rechte</h2>
      <p>
        Für Auskunft, Berichtigung, Löschung, Einschränkung der Verarbeitung sowie zur
        Übermittlung Ihrer Daten wenden Sie sich bitte an{' '}
        <a href="mailto:datenschutz@werkstatt-vogel.de">datenschutz@werkstatt-vogel.de</a> oder
        telefonisch unter 030 123 456 78. Sie haben zudem das Recht, sich bei der zuständigen
        Aufsichtsbehörde zu beschweren (Berliner Beauftragte für Datenschutz und
        Informationsfreiheit).
      </p>

      <h2>Keine Drittanbieter-Ressourcen vor Einwilligung</h2>
      <p>
        Beim Aufruf der Seiten werden keine Ressourcen von Drittanbietern geladen. Schriftarten und
        Skripte werden ausschließlich vom eigenen Ursprung dieses Portals ausgeliefert; es werden
        keine externen Schriftarten, Skripte oder Content-Delivery-Networks eingebunden.
      </p>

      <h2>Protokolle und Fehlermeldungen</h2>
      <p>
        Personenbezogene Daten (Name, E-Mail, Telefon, Kennzeichen) erscheinen weder in
        technischen Protokollen noch in Fehlermeldungen. Fehlerantworten enthalten ausschließlich
        stabile Fehlercodes und eine allgemeine, verständliche Meldung.
      </p>

      <h2>Hinweis zu Benachrichtigungen</h2>
      <p>
        Beim Erstellen einer Rechnung wird eine Benachrichtigung im internen Postausgang abgelegt.
        Ein tatsächlicher E-Mail-Versand über einen SMTP-Dienst findet nicht statt; es werden dabei
        keine Daten an Dritte übermittelt.
      </p>
    </article>
  );
}
