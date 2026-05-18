"use client";

import { createContext, useContext, ReactNode } from "react";

interface Datasource {
  id: string;
  name: string;
  type: string;
}

interface DatasourceContextType {
  datasources: Datasource[];
  getDatasourceName: (id?: string) => string;
}

const DatasourceContext = createContext<DatasourceContextType | undefined>(undefined);

interface DatasourceProviderProps {
  children: ReactNode;
  datasources: Datasource[];
}

export function DatasourceProvider({ children, datasources }: DatasourceProviderProps) {
  const getDatasourceName = (id?: string): string => {
    if (!id) return "-";
    const datasource = datasources.find(ds => ds.id === id);
    return datasource ? datasource.name : "-";
  };

  return (
    <DatasourceContext.Provider value={{ datasources, getDatasourceName }}>
      {children}
    </DatasourceContext.Provider>
  );
}

export function useDatasource() {
  const context = useContext(DatasourceContext);
  if (context === undefined) {
    throw new Error('useDatasource must be used within a DatasourceProvider');
  }
  return context;
}
